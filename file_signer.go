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
	fmt.Printf("Zipping and Signing: %s\n", "Point1")
	dir, err := ioutil.TempDir("", "pass")
	if err != nil {
		return nil, err
	}

	fmt.Printf("Zipping and Signing: %s\n", dir)
	if err := t.ProvisionPassAtDirectory("/Users/gallardo/temp/applepassgen"); err != nil {
		// if err := t.ProvisionPassAtDirectory(dir); err != nil {
		fmt.Printf("Provision Directory error: %s\n", err)
		return nil, err
	}

	fmt.Printf("Zipping and Signing: %s\n", "Point3")
	if err := f.createPassJSONFile(p, dir); err != nil {
		return nil, err
	}

	fmt.Printf("Zipping and Signing: %s\n", "Point4")
	if pz != nil {
		if err := f.createPersonalizationJSONFile(pz, dir); err != nil {
			return nil, err
		}
	}

	fmt.Printf("Zipping and Signing: %s\n", "Point5")
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
	err = ioutil.WriteFile(signatureFileName, signedMfst, 0644)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Zipping and Signing: %s\n", "Point8")
	z, err := f.createZipFile(dir)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Zipping and Signing: %s\n", "Point9")
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

	err = ioutil.WriteFile(manifestJsonFileName, bm, 0644)
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

	err := addFiles(w, tmpDir, "")
	if err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
