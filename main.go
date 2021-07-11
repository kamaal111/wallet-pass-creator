package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
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

	hashedManifest, err := hasher.HashFiles(assetsPath, true)
	if err != nil {
		log.Fatal(err.Error())
	}
	manifest, err := json.MarshalIndent(hashedManifest, "", " ")
	if err != nil {
		log.Fatal(err.Error())
	}
	_ = ioutil.WriteFile("manifest.json", manifest, 0644)
	log.Println("done creating manifest.json")

	// TODO: Get from env
	certificatePassword := fmt.Sprintf("pass:%s", "yes")
	// TODO: Get from env and if empty throw error
	keyPassword := fmt.Sprintf("pass:%s", "y")

	createPasscertificate := exec.Command(OPENSSL_APP, "pkcs12", "-in", "Certificates.p12", "-clcerts", "-nokeys",
		"-out", "passcertificate.pem", "-passin", certificatePassword)
	_, err = createPasscertificate.Output()
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("done creating passcertificate.pem")

	createPasskey := exec.Command(OPENSSL_APP, "pkcs12", "-in", "Certificates.p12", "-nocerts", "-out", "passkey.pem",
		"-passin", certificatePassword, "-passout", keyPassword)
	_, err = createPasskey.Output()
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("done creating passkey.pem")

	createSignature := exec.Command(OPENSSL_APP, "smime", "-binary", "-sign", "-certfile", "Apple Worldwide Developer Relations Certification Authority.pem",
		"-signer", "passcertificate.pem", "-inkey", "passkey.pem", "-in", "manifest.json", "-out", "signature", "-outform",
		"DER", "-passin", keyPassword)
	_, err = createSignature.Output()
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("done creating signature")

	dirFiles, err := ioutil.ReadDir(assetsPath)
	if err != nil {
		log.Fatalln(err)
	}

	for _, file := range dirFiles {
		fileName := file.Name()
		copy(files.AppendFileToPath(assetsPath, fileName), fileName)
	}

	// TODO: Get file names of assets from manifest.json
	filesToZip := []string{
		"manifest.json",
		"pass.json",
		"signature",
		"logo.png",
		"logo@2x.png",
		"icon.png",
		"icon@2x.png",
		"thumbnail.png",
		"thumbnail@2x.png",
	}
	pkPassOutput := fmt.Sprintf("%s.pkpass", passName)
	err = zipFiles(pkPassOutput, filesToZip)
	if err != nil {
		cleanUp(dirFiles, assetsPath)
		log.Fatalln(err)
	}
	cleanUp(dirFiles, assetsPath)

	openCommand := exec.Command("open", pkPassOutput)
	_, err = openCommand.Output()
	if err != nil {
		log.Fatalln(err)
	}

	timeElapsed := time.Since(startTimer)
	log.Printf("all done creating %s in %s\n", pkPassOutput, timeElapsed)

}

func cleanUp(dirFiles []fs.FileInfo, assetsPath string) {
	for _, file := range dirFiles {
		fileName := file.Name()
		copy(fileName, files.AppendFileToPath(assetsPath, fileName))
	}
	log.Println("cleaning up the mess I made")
}

func copy(fromPath string, destination string) error {
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
