package security

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/oceano-dev/microservices-go-common/config"
	"github.com/oceano-dev/microservices-go-common/helpers"
	"github.com/oceano-dev/microservices-go-common/services"

	"github.com/eapache/go-resiliency/breaker"
)

type ManagerCertificates struct {
	config  *config.Config
	service services.CertificateService
}

var (
	certPath string
	keyPath  string
)

func NewManagerCertificates(
	config *config.Config,
	service services.CertificateService,
) *ManagerCertificates {
	certPath = fmt.Sprintf("certs/%s.crt", config.Certificates.FileName)
	keyPath = fmt.Sprintf("certs/%s.key", config.Certificates.FileName)
	return &ManagerCertificates{
		config:  config,
		service: service,
	}
}

func (m *ManagerCertificates) VerifiyLocalCertificateIsValid() bool {
	if helpers.FileExists(certPath) && helpers.FileExists(keyPath) {
		cert, err := m.readCertificate(certPath)
		if err != nil {
			return false
		}

		if cert == nil || cert.NotAfter.AddDate(0, 0, -7).Sub(time.Now().UTC()) <= 0 {
			return false
		}

		return true
	}

	return false
}

func (m *ManagerCertificates) GetCertificate() error {
	err := m.refreshCertificate()
	if err != nil {
		return err
	}

	return nil
}

func (m *ManagerCertificates) GetPathsCertificateAndKey() (string, string) {
	return certPath, keyPath
}

func (m *ManagerCertificates) refreshCertificate() error {
	err := m.requestCertificate()
	if err != nil {
		return err
	}

	err = m.requestCertificateKey()
	if err != nil {
		return err
	}

	return nil
}

func (m *ManagerCertificates) requestCertificate() error {
	b := breaker.New(3, 1, 5*time.Second)
	for {
		var cert []byte
		var err error
		err = b.Run(func() error {
			cert, err = m.service.GetCertificate()
			if err != nil {
				return err
			}

			return nil
		})

		switch err {
		case nil:
			if cert == nil {
				return errors.New("certificate not found")
			}

			err := m.createFile(cert, certPath)
			if err != nil {
				return err
			}

			return nil
		case breaker.ErrBreakerOpen:
			return err
		}
	}
}

func (m *ManagerCertificates) requestCertificateKey() error {
	b := breaker.New(3, 1, 5*time.Second)
	for {
		var key []byte
		var err error
		err = b.Run(func() error {
			key, err = m.service.GetCertificateKey()
			if err != nil {
				return err
			}

			return nil
		})

		switch err {
		case nil:
			if key == nil {
				return errors.New("certificate key not found")
			}

			err := m.createFile(key, keyPath)
			if err != nil {
				return err
			}

			return nil
		case breaker.ErrBreakerOpen:
			return err
		}
	}
}

func (m *ManagerCertificates) getCertificateKey() ([]byte, error) {
	key, err := m.service.GetCertificateKey()
	if err != nil {
		return nil, err
	}

	return key, nil
}

func (m *ManagerCertificates) createFile(filePEM []byte, pathFile string) error {
	file, err := os.Create(pathFile)
	if err != nil {
		os.Exit(1)
		return errors.New("invalid file path")
	}

	_, err = file.Write(filePEM)
	if err != nil {
		os.Exit(1)
		return fmt.Errorf("error when write file %s: %s \n", pathFile, err)
	}

	return nil
}

func (m *ManagerCertificates) readCertificate(pathCertificate string) (*x509.Certificate, error) {
	data, err := ioutil.ReadFile(pathCertificate)
	if err != nil {
		os.Exit(1)
		return nil, fmt.Errorf("read Certificate file error")
	}

	pemBlock, _ := pem.Decode(data)
	if pemBlock == nil {
		return nil, fmt.Errorf("decode Certificate error")
	}

	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}

	return cert, nil
}