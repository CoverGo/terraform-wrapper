package tfinstall

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"net/http"

	"golang.org/x/crypto/openpgp"
)

func ensureInstallDir(installDir string) (string, error) {
	if installDir == "" {
		return ioutil.TempDir("", "tfexec")
	}

	if _, err := os.Stat(installDir); err != nil {
		return "", fmt.Errorf("could not access directory %s for installing Terraform: %w", installDir, err)
	}

	return installDir, nil
}

func unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

func downloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func downloadWithVerification(ctx context.Context, tfVersion string, installDir string, appendUserAgent string) (string, error) {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	// setup: ensure we have a place to put our downloaded terraform binary
	tfDir, err := ensureInstallDir(installDir)
	if err != nil {
		return "", err
	}

	// firstly, download and verify the signature of the checksum file

	sumsTmpDir, err := ioutil.TempDir("", "tfinstall")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(sumsTmpDir)

	sumsFilename := "terraform_" + tfVersion + "_SHA256SUMS"
	sumsSigFilename := sumsFilename + ".72D7468F.sig"

	sumsFullpath := fmt.Sprintf("%s/%s", sumsTmpDir, sumsFilename)
	sumsSigFullpath := fmt.Sprintf("%s/%s", sumsTmpDir, sumsSigFilename)

	sumsURL := fmt.Sprintf("%s/%s/%s", baseURL, tfVersion, sumsFilename)
	sumsSigURL := fmt.Sprintf("%s/%s/%s", baseURL, tfVersion, sumsSigFilename)

	fmt.Println(sumsFullpath)

	if err := downloadFile(sumsFullpath, sumsURL); err != nil {
		return "", fmt.Errorf("error fetching checksums at URL %s: %w", sumsURL, err)
	}

	if err := downloadFile(sumsSigFullpath, sumsSigURL); err != nil {
		return "", fmt.Errorf("error fetching checksums signature: %s", err)
	}

	sumsPath := filepath.Join(sumsTmpDir, sumsFilename)
	sumsSigPath := filepath.Join(sumsTmpDir, sumsSigFilename)

	err = verifySumsSignature(sumsPath, sumsSigPath)
	if err != nil {
		return "", err
	}

	// secondly, download Terraform itself, verifying the checksum
	url := tfURL(tfVersion, osName, archName)

	terraformZipPath := filepath.Join(tfDir, "terraform.zip")

	if err := downloadFile(terraformZipPath, url); err != nil {
		return "", err
	}

	if _, err := unzip(terraformZipPath, tfDir); err != nil {
		return "", err
	}

	return filepath.Join(tfDir, "terraform"), nil
}

// verifySumsSignature downloads SHA256SUMS and SHA256SUMS.sig and verifies
// the signature using the HashiCorp public key.
func verifySumsSignature(sumsPath, sumsSigPath string) error {
	el, err := openpgp.ReadArmoredKeyRing(strings.NewReader(hashicorpPublicKey))
	if err != nil {
		return err
	}
	data, err := os.Open(sumsPath)
	if err != nil {
		return err
	}
	sig, err := os.Open(sumsSigPath)
	if err != nil {
		return err
	}
	_, err = openpgp.CheckDetachedSignature(el, data, sig)

	return err
}

func tfURL(tfVersion, osName, archName string) string {
	sumsFilename := "terraform_" + tfVersion + "_SHA256SUMS"
	sumsURL := fmt.Sprintf("%s/%s/%s", baseURL, tfVersion, sumsFilename)
	return fmt.Sprintf(
		"%s/%s/terraform_%s_%s_%s.zip?checksum=file:%s",
		baseURL, tfVersion, tfVersion, osName, archName, sumsURL,
	)
}
