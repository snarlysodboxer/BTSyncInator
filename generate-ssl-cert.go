package main

import (
  "crypto/rand"
  "crypto/rsa"
  "crypto/x509"
  "crypto/x509/pkix"
  "encoding/pem"
  "log"
  "math/big"
  "os"
  "time"
)

func genCACert(caName string, years int) {
  priv, err := rsa.GenerateKey(rand.Reader, 2048)
  if err != nil {
    log.Fatalf("failed to generate private key: %s", err)
    return
  }

  now := time.Now()

  template := x509.Certificate{
    BasicConstraintsValid: true,
    SerialNumber: new(big.Int).SetInt64(0),
    Subject: pkix.Name{
      CommonName:   os.Getenv("HOSTNAME"),
      Organization: []string{caName},
    },
    NotBefore:    now.Add(-5 * time.Minute).UTC(),
    NotAfter:     now.AddDate(years, 0, 0).UTC(), // valid for years
    IsCA:         true,
    SubjectKeyId: []byte{1, 2, 3, 4},
    KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
  }

  derBytes, err := x509.CreateCertificate(
    rand.Reader, &template, &template, &priv.PublicKey, priv)
  if err != nil {
    log.Fatalf("Failed to create CA Certificate: %s", err)
    return
  }

  certOut, err := os.Create(caName + ".crt")
  if err != nil {
    log.Fatalf("Failed to open "+caName+".crt for writing: %s", err)
    return
  }
  pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
  certOut.Close()
  log.Print("Written " + caName + ".crt\n")

  keyOut, err := os.OpenFile(caName+".key", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
  if err != nil {
    log.Print("Failed to open "+caName+".key for writing:", err)
    return
  }
  pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY",
    Bytes: x509.MarshalPKCS1PrivateKey(priv)})
  keyOut.Close()
  log.Print("Written " + caName + ".key\n")
}
