package git

import (

	"fmt"

	"io" // new import

	"io/fs"

	"os"

	"path/filepath"

	"time"



	"github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/plumbing/object"

)



type Commit struct {

	Hash    string

	Message string

	Author  string

	Date    time.Time

}



type FileInfo struct {



	Path string



	Size int64



}



// InitRepo creates a new git repository

func InitRepo(path string) error {

	if err := os.MkdirAll(path, 0755); err != nil {

		return fmt.Errorf("failed to create directory %s: %w", path, err)

	}



	_, err := git.PlainInit(path, false)

	if err != nil {

		return fmt.Errorf("failed to initialize git repository at %s: %w", path, err)

	}

	return nil

}



// AddAll stages all changes

func AddAll(repoPath string) error {

	r, err := git.PlainOpen(repoPath)

	if err != nil {

		return fmt.Errorf("failed to open repository at %s: %w", repoPath, err)

	}



	w, err := r.Worktree()

	if err != nil {

		return fmt.Errorf("failed to get worktree for repository at %s: %w", repoPath, err)

	}



	_, err = w.Add(".")

	if err != nil {

		return fmt.Errorf("failed to add all files to staging: %w", err)

	}

	return nil

}



// Commit creates a new commit

func CommitChanges(repoPath, author, message string) (*Commit, error) {

	r, err := git.PlainOpen(repoPath)

	if err != nil {

		return nil, fmt.Errorf("failed to open repository at %s: %w", repoPath, err)

	}



	w, err := r.Worktree()

	if err != nil {

		return nil, fmt.Errorf("failed to get worktree for repository at %s: %w", repoPath, err)

	}



	// Add all changes

	if err := AddAll(repoPath); err != nil {

		return nil, err

	}



	// Check if there are any changes to commit

	status, err := w.Status()

	if err != nil {

		return nil, fmt.Errorf("failed to get worktree status: %w", err)

	}



	// If there are no changes, return the last commit (noop)

	if status.IsClean() {

		ref, err := r.Head()

		if err != nil {

			return nil, fmt.Errorf("failed to get HEAD reference: %w", err)

		}



		obj, err := r.CommitObject(ref.Hash())

		if err != nil {

			return nil, fmt.Errorf("failed to get last commit object: %w", err)

		}



		return &Commit{

			Hash:    obj.Hash.String(),

			Message: obj.Message,

			Author:  obj.Author.String(),

			Date:    obj.Author.When,

		}, nil

	}



	// Create commit

	commitHash, err := w.Commit(message, &git.CommitOptions{

		Author: &object.Signature{

			Name:  author,

			Email: author, // Assuming author is also the email for simplicity

			When:  time.Now(),

		},

	})

	if err != nil {

		return nil, fmt.Errorf("failed to commit changes: %w", err)

	}



	// Get the commit object

	obj, err := r.CommitObject(commitHash)

	if err != nil {

		return nil, fmt.Errorf("failed to get commit object: %w", err)

	}



	return &Commit{

		Hash:    obj.Hash.String(),

		Message: obj.Message,

		Author:  obj.Author.String(), // This will include name and email

		Date:    obj.Author.When,

	}, nil

}



// GetHistory returns commit history

func GetHistory(repoPath string) ([]*Commit, error) {

	r, err := git.PlainOpen(repoPath)

	if err != nil {

		return nil, fmt.Errorf("failed to open repository at %s: %w", repoPath, err)

	}



	cIter, err := r.Log(&git.LogOptions{Order: git.LogOrderCommitterTime})

	if err != nil {

		return nil, fmt.Errorf("failed to get commit log: %w", err)

	}



	var commits []*Commit

	err = cIter.ForEach(func(c *object.Commit) error {

		commits = append(commits, &Commit{

			Hash:    c.Hash.String(),

			Message: c.Message,

			Author:  c.Author.String(),

			Date:    c.Author.When,

		})

		return nil

	})

	if err != nil {

		return nil, fmt.Errorf("failed to iterate commits: %w", err)

	}



	return commits, nil

}



// ErrGitFileNotFound is returned when a requested file does not exist in the git repository.



var ErrGitFileNotFound = fmt.Errorf("file not found in git repository")







// ReadFile reads a file from the repository







func ReadFile(repoPath, filePath string) (io.ReadCloser, int64, error) {







	r, err := git.PlainOpen(repoPath)







	if err != nil {







		return nil, 0, fmt.Errorf("failed to open repository at %s: %w", repoPath, err)







	}















	w, err := r.Worktree()







	if err != nil {







		return nil, 0, fmt.Errorf("failed to get worktree for repository at %s: %w", repoPath, err)







	}















		file, err := w.Filesystem.Open(filePath)















		if err != nil {















			if os.IsNotExist(err) {















				return nil, 0, ErrGitFileNotFound















			}















			return nil, 0, fmt.Errorf("failed to open file %s: %w", filePath, err)















		}















	















		fileInfo, err := w.Filesystem.Stat(filePath) // Use w.Filesystem.Stat















		if err != nil {















			file.Close() // Ensure file is closed on error















			return nil, 0, fmt.Errorf("failed to get file info for %s: %w", filePath, err)















		}















	















		return file, fileInfo.Size(), nil















	}



// WriteFile writes a file to the repository

func WriteFile(repoPath, filePath, content string) error {

	fullPath := filepath.Join(repoPath, filePath)



	// Create directory if it doesn't exist

	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0755); err != nil {

		return fmt.Errorf("failed to create directory %s: %w", dir, err)

	}



	return os.WriteFile(fullPath, []byte(content), 0644)

}



// DeleteFile removes a file from the repository

func DeleteFile(repoPath, filePath string) error {

	fullPath := filepath.Join(repoPath, filePath)

	return os.Remove(fullPath)

}



// ListFiles lists all files in the repository (excluding .git)



func ListFiles(repoPath string) ([]*FileInfo, error) {







	// We are using filepath.WalkDir to list files on the filesystem, not the git index.



	// We don't need to open the git repository here.







	var files []*FileInfo







	err := filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {



		if err != nil {



			return err



		}







		// Skip .git directory



		if d.IsDir() && d.Name() == ".git" {



			return fs.SkipDir



		}







		// Skip directories



		if d.IsDir() {



			return nil



		}







		// Get relative path



		relPath, err := filepath.Rel(repoPath, path)



		if err != nil {



			return err



		}







		info, err := d.Info()



		if err != nil {



			return err



		}







		files = append(files, &FileInfo{



			Path: relPath,



			Size: info.Size(),



		})







		return nil



	})







	if err != nil {



		return nil, fmt.Errorf("failed to walk directory %s: %w", repoPath, err)



	}







	return files, nil



}
