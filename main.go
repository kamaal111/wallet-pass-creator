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

	CERTIFICATE_PASSWORD := os.Getenv("CERTIFICATE_PASSWORD")
	KEY_PASSWORD := os.Getenv("KEY_PASSWORD")
	WWDR_PATH := os.Getenv("WWDR_PATH")
	CERTIFICATE_PATH := os.Getenv("CERTIFICATE_PATH")
	PK_PASS_NAME := os.Getenv("PK_PASS_NAME")
	CLEAN_UP := os.Getenv("CLEAN_UP")

	if KEY_PASSWORD == "" {
		log.Fatalln("KEY_PASSWORD is required to use this script")
	}
	if WWDR_PATH == "" {
		defaultWWDRPath := "Apple Worldwide Developer Relations Certification Authority.pem"
		log.Printf("no WWDR_PATH has been provided, will use default of %s\n", defaultWWDRPath)
		WWDR_PATH = defaultWWDRPath
	}
	if CERTIFICATE_PATH == "" {
		defaultCertificatePath := "Certificates.p12"
		log.Printf("no CERTIFICATE_PATH has been provided, will use default of %s\n", defaultCertificatePath)
		CERTIFICATE_PATH = defaultCertificatePath
	}
	if PK_PASS_NAME == "" {
		log.Fatalln("PK_PASS_NAME is required to use this script")
	}
	if CLEAN_UP == "" {
		defaultCleanUp := "Yes"
		log.Printf("no CLEAN_UP has been provided, will use default of %s\n", defaultCleanUp)
		CLEAN_UP = defaultCleanUp
	}

	cleanUpBool := contains([]string{"yes", "Yes", "y", "Y"}, CLEAN_UP)

	assetsPath := fmt.Sprintf("%s.pass/", PK_PASS_NAME)
	_, err := os.Stat(assetsPath)
	if os.IsNotExist(err) {
		log.Fatalf("make sure to set all your assets in %s\n", assetsPath)
	}

	err = createManifestJSON(assetsPath)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("done creating manifest.json")

	certificatePassword := fmt.Sprintf("pass:%s", CERTIFICATE_PASSWORD)
	err = createPasscertificate(certificatePassword, CERTIFICATE_PATH)
	if err != nil {
		log.Println("something went wrong while creating passcertificate.pem, probably an wrong CERTIFICATE_PATH or CERTIFICATE_PASSWORD")
		log.Fatalln(err)
	}
	log.Println("done creating passcertificate.pem")

	keyPassword := fmt.Sprintf("pass:%s", KEY_PASSWORD)

	err = createPasskey(certificatePassword, keyPassword, CERTIFICATE_PATH)
	if err != nil {
		log.Println("something went wrong while creating passkey.pem, probably an wrong KEY_PASSWORD")
		log.Fatalln(err)
	}
	log.Println("done creating passkey.pem")

	err = createSignature(keyPassword, WWDR_PATH)
	if err != nil {
		log.Println("something went wrong while creating signature, probably an wrong WWDR_PATH")
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
	pkPassOutput := fmt.Sprintf("%s.pkpass", PK_PASS_NAME)
	err = zipFiles(pkPassOutput, filesToZip)

	var filesToCleanUp []string
	if cleanUpBool {
		filesToCleanUp = append(generatedFilesToZip, []string{
			"passcertificate.pem",
			"passkey.pem",
		}...)
		log.Println("cleaning up passcertificate.pem and passkey.pem")
	} else {
		filesToCleanUp = generatedFilesToZip
	}

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

func createPasscertificate(certificatePassword string, certificatePath string) error {
	command := exec.Command(OPENSSL_APP, "pkcs12", "-in", certificatePath, "-clcerts", "-nokeys",
		"-out", "passcertificate.pem", "-passin", certificatePassword)
	_, err := command.Output()
	return err
}

func createPasskey(certificatePassword string, keyPassword string, certificatePath string) error {
	createPasskey := exec.Command(OPENSSL_APP, "pkcs12", "-in", certificatePath, "-nocerts", "-out", "passkey.pem",
		"-passin", certificatePassword, "-passout", keyPassword)
	_, err := createPasskey.Output()
	return err
}

func createSignature(keyPassword string, wwdrPath string) error {
	createSignature := exec.Command(OPENSSL_APP, "smime", "-binary", "-sign", "-certfile", wwdrPath,
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

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
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
