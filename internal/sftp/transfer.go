package sftp

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/sftp"
)

type TransferProgress struct {
	Path  string
	Bytes int64
	Total int64
	Done  bool
	Err   error
}

func UploadRecursive(ctx context.Context, client *sftp.Client, localPath, remotePath string, workers int, progress func(TransferProgress)) error {
	if workers < 1 {
		workers = 4
	}
	info, err := os.Stat(localPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return uploadOne(ctx, client, localPath, remotePath, progress)
	}
	files := make(chan string)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range files {
				rel, _ := filepath.Rel(localPath, p)
				dst := remotePath
				if rel != "." {
					dst = strings.ReplaceAll(filepath.Join(remotePath, rel), "\\", "/")
				}
				if e := uploadOne(ctx, client, p, dst, progress); e != nil {
					select {
					case errCh <- e:
					case <-ctx.Done():
					}
					return
				}
			}
		}()
	}
	walkErr := filepath.Walk(localPath, func(p string, fi os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if e = ctx.Err(); e != nil {
			return e
		}
		if fi.IsDir() {
			rel, _ := filepath.Rel(localPath, p)
			if rel != "." {
				_ = client.MkdirAll(strings.ReplaceAll(filepath.Join(remotePath, rel), "\\", "/"))
			}
			return nil
		}
		select {
		case files <- p:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	close(files)
	wg.Wait()
	select {
	case e := <-errCh:
		return e
	default:
		return walkErr
	}
}

func uploadOne(ctx context.Context, client *sftp.Client, localPath, remotePath string, progress func(TransferProgress)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fi, err := os.Stat(localPath)
	if err != nil {
		return err
	}
	if err := client.MkdirAll(filepath.ToSlash(filepath.Dir(remotePath))); err != nil {
		return err
	}
	in, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := client.Create(remotePath)
	if err != nil {
		return err
	}
	defer out.Close()
	buf := make([]byte, 128*1024)
	var copied int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, rerr := in.Read(buf)
		if n > 0 {
			wn, werr := out.Write(buf[:n])
			copied += int64(wn)
			if progress != nil {
				progress(TransferProgress{Path: localPath, Bytes: copied, Total: fi.Size()})
			}
			if werr != nil {
				return werr
			}
			if wn != n {
				return io.ErrShortWrite
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	if progress != nil {
		progress(TransferProgress{Path: localPath, Bytes: copied, Total: fi.Size(), Done: true})
	}
	return nil
}

func DownloadRecursive(ctx context.Context, client *sftp.Client, remotePath, localPath string, workers int, progress func(TransferProgress)) error {
	info, err := client.Stat(remotePath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return downloadOne(ctx, client, remotePath, localPath, progress)
	}
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return err
	}
	entries, err := client.ReadDir(remotePath)
	if err != nil {
		return err
	}
	if workers < 1 {
		workers = 4
	}
	jobs := make(chan [2]string)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				src, dst := j[0], j[1]
				fi, e := client.Stat(src)
				if e != nil {
					select {
					case errCh <- e:
					case <-ctx.Done():
					}
					return
				}
				if fi.IsDir() {
					e = DownloadRecursive(ctx, client, src, dst, 1, progress)
				} else {
					e = downloadOne(ctx, client, src, dst, progress)
				}
				if e != nil {
					select {
					case errCh <- e:
					case <-ctx.Done():
					}
					return
				}
			}
		}()
	}
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			close(jobs)
			wg.Wait()
			return err
		}
		jobs <- [2]string{strings.TrimRight(remotePath, "/") + "/" + e.Name(), filepath.Join(localPath, e.Name())}
	}
	close(jobs)
	wg.Wait()
	select {
	case e := <-errCh:
		return e
	default:
		return ctx.Err()
	}
}

func downloadOne(ctx context.Context, client *sftp.Client, remotePath, localPath string, progress func(TransferProgress)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}
	in, err := client.Open(remotePath)
	if err != nil {
		return err
	}
	defer in.Close()
	fi, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer out.Close()
	buf := make([]byte, 128*1024)
	var copied int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, rerr := in.Read(buf)
		if n > 0 {
			wn, werr := out.Write(buf[:n])
			copied += int64(wn)
			if progress != nil {
				progress(TransferProgress{Path: remotePath, Bytes: copied, Total: fi.Size()})
			}
			if werr != nil {
				return werr
			}
			if wn != n {
				return io.ErrShortWrite
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	if progress != nil {
		progress(TransferProgress{Path: remotePath, Bytes: copied, Total: fi.Size(), Done: true})
	}
	return nil
}

func (p TransferProgress) String() string {
	if p.Err != nil {
		return fmt.Sprintf("%s: %v", p.Path, p.Err)
	}
	return fmt.Sprintf("%s: %d/%d", p.Path, p.Bytes, p.Total)
}
