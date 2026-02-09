package todo

import (
	"strings"
	"time"

	"github.com/quailyquaily/mistermorph/internal/fsstore"
	"github.com/quailyquaily/mistermorph/internal/pathutil"
)

func NewStore(wipPath string, donePath string) *Store {
	wipPath = pathutil.ExpandHomePath(strings.TrimSpace(wipPath))
	donePath = pathutil.ExpandHomePath(strings.TrimSpace(donePath))
	return &Store{
		WIPPath:  wipPath,
		DONEPath: donePath,
		Now:      time.Now,
	}
}

func (s *Store) readFiles() (WIPFile, DONEFile, error) {
	now := s.nowUTC()
	wip, err := s.readWIP(now)
	if err != nil {
		return WIPFile{}, DONEFile{}, err
	}
	done, err := s.readDONE(now)
	if err != nil {
		return WIPFile{}, DONEFile{}, err
	}
	return wip, done, nil
}

func (s *Store) writeFiles(wip WIPFile, done DONEFile) error {
	now := s.nowUTC().Format(time.RFC3339)
	if strings.TrimSpace(wip.CreatedAt) == "" {
		wip.CreatedAt = now
	}
	if strings.TrimSpace(done.CreatedAt) == "" {
		done.CreatedAt = now
	}
	wip.UpdatedAt = now
	done.UpdatedAt = now
	wip.OpenCount = len(wip.Entries)
	done.DoneCount = len(done.Entries)
	if err := fsstore.WriteTextAtomic(s.WIPPath, RenderWIP(wip), fsstore.FileOptions{DirPerm: 0o700, FilePerm: 0o600}); err != nil {
		return err
	}
	if err := fsstore.WriteTextAtomic(s.DONEPath, RenderDONE(done), fsstore.FileOptions{DirPerm: 0o700, FilePerm: 0o600}); err != nil {
		return err
	}
	return nil
}

func (s *Store) readWIP(now time.Time) (WIPFile, error) {
	nowRFC3339 := now.UTC().Format(time.RFC3339)
	text, exists, err := fsstore.ReadText(s.WIPPath)
	if err != nil {
		return WIPFile{}, err
	}
	if !exists || strings.TrimSpace(text) == "" {
		return WIPFile{
			CreatedAt: nowRFC3339,
			UpdatedAt: nowRFC3339,
			OpenCount: 0,
			Entries:   nil,
		}, nil
	}
	wip, err := ParseWIP(text)
	if err != nil {
		return WIPFile{}, err
	}
	if strings.TrimSpace(wip.CreatedAt) == "" {
		wip.CreatedAt = nowRFC3339
	}
	if strings.TrimSpace(wip.UpdatedAt) == "" {
		wip.UpdatedAt = nowRFC3339
	}
	wip.OpenCount = len(wip.Entries)
	return wip, nil
}

func (s *Store) readDONE(now time.Time) (DONEFile, error) {
	nowRFC3339 := now.UTC().Format(time.RFC3339)
	text, exists, err := fsstore.ReadText(s.DONEPath)
	if err != nil {
		return DONEFile{}, err
	}
	if !exists || strings.TrimSpace(text) == "" {
		return DONEFile{
			CreatedAt: nowRFC3339,
			UpdatedAt: nowRFC3339,
			DoneCount: 0,
			Entries:   nil,
		}, nil
	}
	done, err := ParseDONE(text)
	if err != nil {
		return DONEFile{}, err
	}
	if strings.TrimSpace(done.CreatedAt) == "" {
		done.CreatedAt = nowRFC3339
	}
	if strings.TrimSpace(done.UpdatedAt) == "" {
		done.UpdatedAt = nowRFC3339
	}
	done.DoneCount = len(done.Entries)
	return done, nil
}

func (s *Store) nowUTC() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
