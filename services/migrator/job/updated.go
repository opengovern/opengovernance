package job

import (
	"crypto/sha256"
	"encoding/base64"
	"github.com/kaytu-io/kaytu-engine/services/migrator/job/types"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

func (w *Job) CheckIfUpdateIsNeeded(name string, mig types.Migration) (bool, error) {
	m, err := w.db.GetMigration(name)
	if err != nil {
		return false, err
	}

	if mig.IsGitBased() {
		return m.AdditionalInfo != w.commit, nil
	} else {
		hashes, err := w.FindFilesHashes(mig)
		if err != nil {
			return false, err
		}

		return m.AdditionalInfo != hashes, nil
	}
}

func (w *Job) UpdateMigration(name string, mig types.Migration) error {
	if mig.IsGitBased() {
		return w.db.UpdateMigrationAdditionalInfo(name, w.commit)
	} else {
		hashes, err := w.FindFilesHashes(mig)
		if err != nil {
			return err
		}

		return w.db.UpdateMigrationAdditionalInfo(name, hashes)
	}
}

func (w *Job) FindFilesHashes(mig types.Migration) (string, error) {
	var fileHashes []types.FileHash
	err := filepath.Walk(mig.AttachmentFolderPath(), func(path string, info fs.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		h := sha256.New()
		hash := h.Sum(content)

		fileHashes = append(fileHashes, types.FileHash{
			Filename: info.Name(),
			Hash:     hash,
		})
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(fileHashes, func(i, j int) bool {
		return fileHashes[i].Filename < fileHashes[j].Filename
	})
	var hashes []byte
	for _, fh := range fileHashes {
		hashes = append(hashes, fh.Hash...)
	}
	return base64.StdEncoding.EncodeToString(hashes), nil
}
