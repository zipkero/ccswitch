package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// 도메인 sentinel error. CLI 경계가 errors.Is로 판별해 종료 코드로 옮긴다.
var (
	// ErrAlreadyExists는 이미 등록된 이름으로 Add를 호출했을 때 돌아온다.
	ErrAlreadyExists = errors.New("profile already exists")
	// ErrStoreCorrupt는 등록 정보 파일을 읽거나 해석할 수 없을 때 돌아온다. 빈 목록으로
	// 간주하지 않고 실패로 구분한다.
	ErrStoreCorrupt = errors.New("profile store is corrupt")
	// ErrNotFound는 등록되지 않은 이름으로 Remove를 호출했을 때 돌아온다.
	ErrNotFound = errors.New("profile not found")
)

// metadataVersion은 지금 이 코드가 읽고 쓸 수 있는 유일한 스키마 버전이다. 다른 값은
// 추측해서 읽지 않고 거부한다.
const metadataVersion = 1

type storeFile struct {
	Version  int              `json:"version"`
	Profiles []storeFileEntry `json:"profiles"`
}

type storeFileEntry struct {
	Name string `json:"name"`
}

// Store는 등록된 프로필 이름의 집합이다. 디렉토리 경로는 담지 않는다 — 경로는 항상
// Layout에서 이름으로부터 다시 계산된다.
type Store struct {
	path  string
	names map[string]struct{}
}

// Load는 path의 등록 정보를 읽는다. 파일이 없는 것은 오류가 아니라 빈 등록으로 취급한다.
// 읽기·파싱에 실패하거나 버전을 모르면 ErrStoreCorrupt로 실패한다.
func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Store{path: path, names: map[string]struct{}{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, ErrStoreCorrupt)
	}

	var sf storeFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("%s: %w", path, ErrStoreCorrupt)
	}
	if sf.Version != metadataVersion {
		return nil, fmt.Errorf("%s: %w", path, ErrStoreCorrupt)
	}

	names := make(map[string]struct{}, len(sf.Profiles))
	for _, p := range sf.Profiles {
		names[p.Name] = struct{}{}
	}
	return &Store{path: path, names: names}, nil
}

// Names는 등록된 이름을 사전순으로 돌려준다.
func (s *Store) Names() []string {
	names := make([]string, 0, len(s.names))
	for name := range s.names {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Has는 name이 등록되어 있는지 본다.
func (s *Store) Has(name string) bool {
	_, ok := s.names[name]
	return ok
}

// Add는 name을 등록에 더한다. 이미 있으면 ErrAlreadyExists를 돌려주고 아무것도 바꾸지 않는다.
func (s *Store) Add(name string) error {
	if s.Has(name) {
		return &alreadyExistsError{
			msg: fmt.Sprintf("%q is already registered; choose a different profile name", name),
		}
	}
	s.names[name] = struct{}{}
	return nil
}

// alreadyExistsError는 사용자에게 보이는 메시지와 errors.Is 분류용 sentinel(ErrAlreadyExists)을
// 나눠 갖는다 — name.go의 nameError, create.go의 dirOccupiedError와 같은 이유로, %w로 감싸면
// sentinel 자신의 문구("profile already exists")가 메시지 끝에 다시 붙어 조치 안내와 겹친다.
type alreadyExistsError struct {
	msg string
}

func (e *alreadyExistsError) Error() string { return e.msg }
func (e *alreadyExistsError) Unwrap() error { return ErrAlreadyExists }

// Remove는 등록에서 name을 지운다. 등록되어 있지 않으면 ErrNotFound를 돌려주고 아무것도
// 바꾸지 않는다.
func (s *Store) Remove(name string) error {
	if !s.Has(name) {
		return &notFoundError{
			msg: fmt.Sprintf("%q is not a registered profile", name),
		}
	}
	delete(s.names, name)
	return nil
}

// notFoundError는 alreadyExistsError·dirOccupiedError·nameError와 같은 이유로 사용자 메시지와
// errors.Is 분류용 sentinel(ErrNotFound)을 나눠 갖는다.
type notFoundError struct {
	msg string
}

func (e *notFoundError) Error() string { return e.msg }
func (e *notFoundError) Unwrap() error { return ErrNotFound }

// Save는 현재 등록 상태를 path에 쓴다. 같은 디렉토리에 매번 새 이름의 임시 파일을 쓰고
// rename으로 교체하며, 실패 경로에서는 임시 파일을 지운다 (D11) — 쓰기 도중 중단되어도
// 이전 등록 내용을 통째로 잃지 않기 위해서다.
func (s *Store) Save() error {
	names := s.Names()
	sf := storeFile{Version: metadataVersion, Profiles: make([]storeFileEntry, len(names))}
	for i, name := range names {
		sf.Profiles[i] = storeFileEntry{Name: name}
	}

	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".profiles-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	saved := false
	defer func() {
		if !saved {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return err
	}
	saved = true
	return nil
}
