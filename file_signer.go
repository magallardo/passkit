package passkit

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type fileSigner struct {
}

func NewFileBasedSigner() Signer {
	return &fileSigner{}
}

func (f *fileSigner) CreateSignedAndZippedPassArchive(p *Pass, t PassTemplate, i *SigningInformation) ([]byte, error) {
	return f.CreateSignedAndZippedPersonalizedPassArchive(p, nil, t, i)
}

func (f *fileSigner) CreateSignedAndZippedPersonalizedPassArchive(p *Pass, pz *Personalization, t PassTemplate, i *SigningInformation) ([]byte, error) {

	dir, err := ioutil.TempDir("", "pass")
	if err != nil {
		return nil, err
	}

	fmt.Printf("Zipping and Signing: %s\n", dir)
	// if err := t.ProvisionPassAtDirectory("/Users/gallardo/temp/applepassgen"); err != nil {
	if err := t.ProvisionPassAtDirectory(dir); err != nil {
		fmt.Printf("Provision Directory error: %s\n", err)
		return nil, err
	}

	if err := f.createPassJSONFile(p, dir); err != nil {
		return nil, err
	}

	if pz != nil {
		if err := f.createPersonalizationJSONFile(pz, dir); err != nil {
			return nil, err
		}
	}

	mfst, err := f.createManifestJSONFile(dir)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Zipping and Signing: %s\n", "Point6")
	signedMfst, err := signManifestFile(mfst, i)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Zipping and Signing: %s\n", "Point7")
	fmt.Printf("Writing signature file: %s\n", signatureFileName)
	// MAG - Changed to write signature to dir.
	// err = ioutil.WriteFile(signatureFileName, signedMfst, 0644)
	err = ioutil.WriteFile(filepath.Join(dir, signatureFileName), signedMfst, 0644)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Zipping and Signing: %s\n", "Point8")
	fmt.Printf("Creating zip at: %s\n", dir)
	z, err := f.createZipFile(dir)
	if err != nil {
		fmt.Printf("Error ziping file: %s\n", err)
		return nil, err
	}

	//Fail silently
	_ = os.RemoveAll(dir)
	return z, nil
}

func (f *fileSigner) SignManifestFile(manifestJson []byte, i *SigningInformation) ([]byte, error) {
	return signManifestFile(manifestJson, i)
}

func (f *fileSigner) createPassJSONFile(p *Pass, tmpDir string) error {
	if !p.IsValid() {
		return fmt.Errorf("%v", p.GetValidationErrors())
	}

	b, err := p.toJSON()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(tmpDir, passJsonFileName), b, 0644)
}

func (f *fileSigner) createPersonalizationJSONFile(pz *Personalization, tmpDir string) error {
	b, err := pz.toJSON()
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(tmpDir, personalizationJsonFileName), b, 0644)
}

func (f *fileSigner) createManifestJSONFile(tmpDir string) ([]byte, error) {
	m, err := f.hashFiles(tmpDir)
	if err != nil {
		return nil, err
	}

	bm, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	// MAG - Changed to writed manifest file to tmpDir
	// err = ioutil.WriteFile(manifestJsonFileName, bm, 0644)
	err = ioutil.WriteFile(filepath.Join(tmpDir, manifestJsonFileName), bm, 0644)
	if err != nil {
		return nil, err
	}

	return bm, nil
}

func (f *fileSigner) hashFiles(tmpDir string) (map[string]string, error) {
	files, err := loadDir(tmpDir)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]string)
	for name, data := range files {
		hash := sha1.Sum(data)
		ret[filepath.Base(name)] = fmt.Sprintf("%x", hash)
	}

	return ret, nil
}

func (f *fileSigner) createZipFile(tmpDir string) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	fmt.Printf("Adding file to zip from: %s\n", tmpDir)
	err := addFiles(w, tmpDir, "")
	if err != nil {
		fmt.Printf("Error adding files: %s\n", err)
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
