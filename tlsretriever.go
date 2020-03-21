package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"path/filepath"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/auth"
	"github.com/Azure/go-autorest/autorest/azure"
	"golang.org/x/crypto/pkcs12"
)

// AzureKeyVaultCertificate can extract certificates (e.g. TLS certificates) stored in Azure Key Vault
type AzureKeyVaultCertificate struct {
	Ctx       context.Context
	VaultName string

	Client keyvault.BaseClient

	authenticated bool
	vaultBaseURL  string
}

// GetKeyVaultClient initializes and authenticates the client to interact with Azure Key Vault
func (akv *AzureKeyVaultCertificate) GetKeyVaultClient() (err error) {
	// Create a new client
	akv.Client = keyvault.New()
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}
	akv.Client.Authorizer = authorizer
	akv.authenticated = true

	// Base URL for the vault
	akv.vaultBaseURL = fmt.Sprintf("https://%s.%s", akv.VaultName, azure.PublicCloud.KeyVaultDNSSuffix)

	return nil
}

func (akv *AzureKeyVaultCertificate) requestCertificateVersion(certificateName string) (version string, err error) {
	// List certificate versions
	list, err := akv.Client.GetCertificateVersionsComplete(akv.Ctx, akv.vaultBaseURL, certificateName, nil)
	if err != nil {
		return "", err
	}

	// Iterate through the list and get the last version
	var lastItemDate time.Time
	var lastItemVersion string
	for list.NotDone() {
		// Get element
		item := list.Value()
		// Filter only enabled items
		if *item.Attributes.Enabled {
			// Get the most recent element
			updatedTime := time.Time(*item.Attributes.Updated)
			if lastItemDate.IsZero() || updatedTime.After(lastItemDate) {
				lastItemDate = updatedTime

				// Get the ID
				parts := strings.Split(*item.ID, "/")
				lastItemVersion = parts[len(parts)-1]
			}
		}
		// Iterate to next
		list.Next()
	}

	return lastItemVersion, nil
}

func (akv *AzureKeyVaultCertificate) requestCertificatePFX(certificateName string, certificateVersion string) (key interface{}, cert *x509.Certificate, err error) {
	// The full certificate, including the key, is stored as a secret in Azure Key Vault, encoded as PFX
	pfx, err := akv.Client.GetSecret(akv.Ctx, akv.vaultBaseURL, certificateName, certificateVersion)
	if err != nil {
		return nil, nil, err
	}

	// Response is a Base64-Encoded PFX, with no passphrase
	pfxBytes, err := base64.StdEncoding.DecodeString(*pfx.Value)
	if err != nil {
		return nil, nil, err
	}
	return pkcs12.Decode(pfxBytes, "")
}

// GetCertificate returns the certificate and key from Azure Key Vault, encoded as PEM
func (akv *AzureKeyVaultCertificate) GetCertificate(certificateName string) (certificate []byte, key []byte, err error) {
	// Error if there's no authenticated client yet
	if !akv.authenticated {
		return nil, nil, errors.New("Need to invoke GetKeyVaultClient() first")
	}

	// List certificate versions
	fmt.Printf("Getting certificate version for %s\n", certificateName)
	certificateVersion, err := akv.requestCertificateVersion(certificateName)
	if err != nil {
		return nil, nil, err
	}

	// Request the certificate and key
	fmt.Printf("Getting PFX for %s\n", certificateName)
	pfxKey, pfxCert, err := akv.requestCertificatePFX(certificateName, certificateVersion)
	keyX509, err := x509.MarshalPKCS8PrivateKey(pfxKey)
	if err != nil {
		return nil, nil, err
	}

	// Convert to PEM
	fmt.Printf("Converting to PEM for %s\n", certificateName)
	keyBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyX509,
	}
	var keyPEM bytes.Buffer
	pem.Encode(&keyPEM, keyBlock)

	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: pfxCert.Raw,
	}
	var certPEM bytes.Buffer
	pem.Encode(&certPEM, certBlock)

	return certPEM.Bytes(), keyPEM.Bytes(), nil
}

// Entry point
func main() {
	// Replace this with the name of the Azure Key Vault
	vaultName := "maks-vault-gps-dev"
	// Replace this with the name of the certificate inside the vault
	certificateName := "sunnycertificate"

	ctx := context.Background()

	// Create an object
	certificate := AzureKeyVaultCertificate{
		Ctx:       ctx,
		VaultName: vaultName,
	}

	// Authenticate
	if err := certificate.GetKeyVaultClient(); err != nil {
		fmt.Println("Error", err)
		return
	}

	// Fetch the certificate and key as PEM
	cert, key, err := certificate.GetCertificate(certificateName)
	if err != nil {
		fmt.Println("Error", err)
		return
	}

	// Write the certificates to disk
	filePath, _ := filepath.Abs("./certs/" + "chamscertificate.pem")
	f, _ := os.Create(filePath)
	f.Write(cert)
	path, err := filepath.Abs(filepath.Dir(filePath))
	if  err != nil {
		fmt.Println("Error", err)
		return
	}

	fmt.Println("files path :" , path)
	f.Close()
	
	filePath2, _ := filepath.Abs("./certs/" + "key.pem")
        f, _ = os.Create(filePath2)
	f.Write(key)
	f.Close()

	for true {
	fmt.Println("certificate and key retrieved")
	time.Sleep(time.Second)
	}
}
