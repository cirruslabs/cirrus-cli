package targz

import (
	"archive/tar"
	"bufio"
	"fmt"
	gzip "github.com/klauspost/pgzip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const DEFAULT_BUFFER_SIZE = 1024 * 1024

func Archive(baseFolder string, folderPaths []string, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("error creating %s: %v", dest, err)
	}
	defer out.Close()

	gzipWriter := gzip.NewWriter(out)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	buffer := make([]byte, DEFAULT_BUFFER_SIZE)

	for _, folderPath := range folderPaths {
		if err := archiveSingleFolder(baseFolder, folderPath, tarWriter, buffer); err != nil {
			return err
		}
	}

	return nil
}

func archiveSingleFolder(baseFolder string, folderPath string, tarWriter *tar.Writer, buffer []byte) error {
	return filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error walking folder %s: %v", path, err)
		}

		header, err := tar.FileInfoHeader(info, path)
		if err != nil {
			return fmt.Errorf("error  making header %s: %v", path, err)
		}
		header.Name = filepath.ToSlash(strings.TrimPrefix(path, baseFolder))

		if header.Typeflag == tar.TypeSymlink {
			linkDest, _ := os.Readlink(path)
			if filepath.IsAbs(linkDest) && strings.HasPrefix(linkDest, baseFolder) {
				linkDest, _ = filepath.Rel(baseFolder, linkDest)
			}
			header.Linkname = filepath.ToSlash(linkDest)
		}

		err = tarWriter.WriteHeader(header)
		if err != nil {
			return fmt.Errorf("%s: writing header: %v", path, err)
		}

		if info.IsDir() {
			return nil
		}

		if header.Typeflag == tar.TypeReg {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("%s: open: %v", path, err)
			}
			defer file.Close()

			_, err = io.CopyBuffer(
				// Work around https://github.com/golang/go/issues/16474
				struct{ io.Writer }{tarWriter},
				// Work around https://github.com/golang/go/issues/16474
				struct{ io.Reader }{file},
				buffer,
			)
			if err != nil && err != io.EOF {
				return fmt.Errorf("%s: copying contents: %v", path, err)
			}
		}
		return nil
	})
}

func Unarchive(tarPath string, destFolder string) error {
	tarFile, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("failed to open tar %s: %v", tarPath, err)
	}
	defer tarFile.Close()

	gzipReader, err := gzip.NewReader(bufio.NewReaderSize(tarFile, DEFAULT_BUFFER_SIZE))
	if err != nil {
		return fmt.Errorf("failed to create new gzip reader %s: %v", tarPath, err)
	}
	defer gzipReader.Close()

	gzipTar := tar.NewReader(gzipReader)

	buffer := make([]byte, DEFAULT_BUFFER_SIZE)

	for {
		header, err := gzipTar.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if err := untarFile(gzipTar, header, destFolder, buffer); err != nil {
			return err
		}
	}
	return nil
}

func untarFile(tr *tar.Reader, header *tar.Header, destination string, buffer []byte) error {
	switch header.Typeflag {
	case tar.TypeDir:
		return mkdir(filepath.Join(destination, filepath.FromSlash(header.Name)))
	case tar.TypeReg, tar.TypeChar, tar.TypeBlock, tar.TypeFifo:
		return writeNewFile(filepath.Join(destination, filepath.FromSlash(header.Name)), tr, header.FileInfo(), buffer)
	case tar.TypeSymlink:
		return writeNewSymbolicLink(filepath.Join(destination, filepath.FromSlash(header.Name)), filepath.FromSlash(header.Linkname))
	case tar.TypeLink:
		return writeNewHardLink(filepath.Join(destination, filepath.FromSlash(header.Name)), filepath.Join(destination, filepath.FromSlash(header.Linkname)))
	default:
		return fmt.Errorf("%s: unknown type flag: %c", header.Name, header.Typeflag)
	}
}

func writeNewFile(fpath string, in io.Reader, fi os.FileInfo, buffer []byte) error {
	err := os.MkdirAll(filepath.Dir(fpath), 0755)
	if err != nil {
		return fmt.Errorf("%s: making directory for file: %v", fpath, err)
	}

	out, err := os.Create(fpath)
	if err != nil {
		return fmt.Errorf("%s: creating new file: %v", fpath, err)
	}
	defer out.Close()

	err = out.Chmod(fi.Mode())
	if err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("%s: changing file mode: %v", fpath, err)
	}

	writtenBytes, err := io.CopyBuffer(
		// Work around https://github.com/golang/go/issues/16474
		struct{ io.Writer }{out},
		io.LimitReader(in, fi.Size()),
		buffer,
	)
	if err != nil {
		return fmt.Errorf("%s: writing file after %d bytes (expected %d): %v", fpath, writtenBytes, fi.Size(), err)
	}
	return nil
}

func writeNewSymbolicLink(fpath string, target string) error {
	err := os.MkdirAll(filepath.Dir(fpath), 0755)
	if err != nil {
		return fmt.Errorf("%s: making directory for file: %v", fpath, err)
	}

	err = os.Symlink(target, fpath)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("%s: making symbolic link for: %v", fpath, err)
	}

	return nil
}

func writeNewHardLink(fpath string, target string) error {
	err := os.MkdirAll(filepath.Dir(fpath), 0755)
	if err != nil {
		return fmt.Errorf("%s: making directory for file: %v", fpath, err)
	}

	err = os.Link(target, fpath)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("%s: making hard link for: %v", fpath, err)
	}

	return nil
}

func mkdir(dirPath string) error {
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return fmt.Errorf("%s: making directory: %v", dirPath, err)
	}
	return nil
}
