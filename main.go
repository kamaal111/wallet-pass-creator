package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/kamaal111/WalletManifestCreator/hasher"
	"github.com/kamaal111/kamaal-go-utils/files"
)

const OPENSSL_APP = "openssl"

func main() {
	startTimer := time.Now()

	passName := "Generic"
	assetsPath := fmt.Sprintf("%s.pass/", passName)

	err := createManifestJSON(assetsPath)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("done creating manifest.json")

	// TODO: Get from env
	certificatePassword := fmt.Sprintf("pass:%s", "yes")

	err = createPasscertificate(certificatePassword)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("done creating passcertificate.pem")

	// TODO: Get from env and if empty throw error
	keyPassword := fmt.Sprintf("pass:%s", "y")

	err = createPasskey(certificatePassword, keyPassword)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("done creating passkey.pem")

	err = createSignature(keyPassword)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("done creating signature")

	assetDirFiles, err := ioutil.ReadDir(assetsPath)
	if err != nil {
		log.Fatalln(err)
	}

	assetFiles := []string{}

	for _, file := range assetDirFiles {
		assetFiles = append(assetFiles, file.Name())
	}

	moveFilesToRoot(assetFiles, assetsPath)

	generatedFilesToZip := []string{
		"manifest.json",
		"signature",
	}
	filesToZip := append(assetFiles, generatedFilesToZip...)
	pkPassOutput := fmt.Sprintf("%s.pkpass", passName)
	err = zipFiles(pkPassOutput, filesToZip)
	filesToCleanUp := append(generatedFilesToZip, []string{
		"passcertificate.pem",
		"passkey.pem",
	}...)
	if err != nil {
		cleanUp(assetFiles, assetsPath, filesToCleanUp)
		log.Fatalln(err)
	}
	cleanUp(assetFiles, assetsPath, filesToCleanUp)

	timeElapsed := time.Since(startTimer)
	log.Printf("all done creating %s in %s\n", pkPassOutput, timeElapsed)

}

func createManifestJSON(assetsPath string) error {
	hashedManifest, err := hasher.HashFiles(assetsPath, true)
	if err != nil {
		return err
	}

	manifest, err := json.MarshalIndent(hashedManifest, "", " ")
	if err != nil {
		return err
	}

	_ = ioutil.WriteFile("manifest.json", manifest, 0644)
	return nil
}

func createPasscertificate(certificatePassword string) error {
	command := exec.Command(OPENSSL_APP, "pkcs12", "-in", "Certificates.p12", "-clcerts", "-nokeys",
		"-out", "passcertificate.pem", "-passin", certificatePassword)
	_, err := command.Output()
	return err
}

func createPasskey(certificatePassword string, keyPassword string) error {
	createPasskey := exec.Command(OPENSSL_APP, "pkcs12", "-in", "Certificates.p12", "-nocerts", "-out", "passkey.pem",
		"-passin", certificatePassword, "-passout", keyPassword)
	_, err := createPasskey.Output()
	return err
}

func createSignature(keyPassword string) error {
	createSignature := exec.Command(OPENSSL_APP, "smime", "-binary", "-sign", "-certfile", "Apple Worldwide Developer Relations Certification Authority.pem",
		"-signer", "passcertificate.pem", "-inkey", "passkey.pem", "-in", "manifest.json", "-out", "signature", "-outform",
		"DER", "-passin", keyPassword)
	_, err := createSignature.Output()
	return err
}

func moveFilesToRoot(assetFiles []string, assetsPath string) {
	for _, file := range assetFiles {
		move(files.AppendFileToPath(assetsPath, file), file)
	}
}

func cleanUp(assetFiles []string, assetsPath string, filesToRemove []string) {
	for _, file := range assetFiles {
		move(file, files.AppendFileToPath(assetsPath, file))
	}

	for _, file := range filesToRemove {
		err := os.Remove(file)
		if err != nil {
			log.Println(err)
		}
	}

	log.Println("cleaning up the mess I made")
}

func move(fromPath string, destination string) error {
	return os.Rename(fromPath, destination)
}

func zipFiles(filename string, files []string) error {
	newZipFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	for _, file := range files {
		if err = addFileToZip(zipWriter, file); err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(zipWriter *zip.Writer, filename string) error {
	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filename
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}
