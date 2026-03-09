package container

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/weiyong1024/clawsandbox/internal/assets"
)

func ImageExists(cli *docker.Client, imageRef string) (bool, error) {
	images, err := cli.ListImages(docker.ListImagesOptions{All: false})
	if err != nil {
		return false, fmt.Errorf("listing images: %w", err)
	}
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imageRef {
				return true, nil
			}
		}
	}
	return false, nil
}

func Build(cli *docker.Client, imageRef string, out io.Writer) error {
	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		buildCtx, err := createBuildContext()
		if err != nil {
			return fmt.Errorf("creating build context: %w", err)
		}

		var logBuf bytes.Buffer
		buildOut := io.MultiWriter(out, &logBuf)

		err = cli.BuildImage(docker.BuildImageOptions{
			Name:           imageRef,
			InputStream:    buildCtx,
			OutputStream:   buildOut,
			RmTmpContainer: true,
		})
		if err == nil {
			return nil
		}

		lastErr = fmt.Errorf("build failed: %w", err)
		if attempt == 2 || !isTransientBuildFailure(logBuf.String(), err) {
			return lastErr
		}

		fmt.Fprintf(out, "\nBuild hit a transient network error. Retrying automatically (%d/2)...\n\n", attempt+1)
		time.Sleep(5 * time.Second)
	}

	return lastErr
}

// TagImage adds an additional tag to an already-built image.
func TagImage(cli *docker.Client, existingRef, repo, tag string) error {
	return cli.TagImage(existingRef, docker.TagImageOptions{
		Repo:  repo,
		Tag:   tag,
		Force: true,
	})
}

// PullImage pulls an image from a remote registry.
func PullImage(cli *docker.Client, repo, tag string, out io.Writer) error {
	return cli.PullImage(docker.PullImageOptions{
		Repository:   repo,
		Tag:          tag,
		OutputStream: out,
	}, docker.AuthConfiguration{})
}

func createBuildContext() (io.Reader, error) {
	buf := &bytes.Buffer{}
	tw := tar.NewWriter(buf)

	err := fs.WalkDir(assets.DockerFS, "docker", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, err := assets.DockerFS.ReadFile(path)
		if err != nil {
			return err
		}
		name := strings.TrimPrefix(path, "docker/")
		mode := int64(0644)
		if name == "entrypoint.sh" {
			mode = 0755
		}
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(content))}); err != nil {
			return err
		}
		_, err = tw.Write(content)
		return err
	})
	if err != nil {
		return nil, err
	}
	tw.Close()
	return buf, nil
}

func isTransientBuildFailure(buildLog string, err error) bool {
	text := strings.ToLower(buildLog)
	if err != nil {
		text += "\n" + strings.ToLower(err.Error())
	}

	for _, marker := range []string{
		"connection failed",
		"temporary failure resolving",
		"i/o timeout",
		"tls handshake timeout",
		"connection reset by peer",
		"unexpected eof",
		"unable to fetch some archives",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}

	return false
}
