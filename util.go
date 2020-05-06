package main

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"strings"

	"crypto/sha1"
)

// storeImages stores base64 images from data into the filesystem
// and replaces the image data with SHA1 checksum in the map
func storeImages(data map[string]interface{}) (err error) {
	var imageHash string

	for k, v := range data {
		if vv, ok := v.(string); ok && k == "coverImageURL" {
			if imageHash, err = storeImage(data, k, vv); err != nil {
				return
			}

			data[k] = imageHash
		} else if vs, ok := v.([]interface{}); ok && k == "imageURLs" {
			var imageHashes []string
			var imageHashesM = make(map[string]bool)

			for _, v := range vs {
				if vv, ok := v.(string); ok {
					if imageHash, err = storeImage(data, k, vv); err != nil {
						return
					}

					if _, ok = imageHashesM[imageHash]; ok {
						continue
					}

					imageHashes = append(imageHashes, imageHash)
				}
			}

			data[k] = imageHashes
		}
	}

	return
}

// Store an base64 image
func storeImage(data map[string]interface{}, k, v string) (imageHash string, err error) {
	if strings.HasPrefix(v, "data:") {
		i := strings.Index(v, ";base64,")
		if i == 0 && i+len(";base64,") < len(v) {
			return
		}

		hash := sha1.New()
		base64Data := v[i+len(";base64,"):]
		hash.Write([]byte(base64Data))

		if err = os.MkdirAll("files", 0700); err != nil {
			return
		}

		var imageData []byte
		if imageData, err = base64.StdEncoding.DecodeString(base64Data); err != nil {
			return
		}

		imageHash = base64.StdEncoding.EncodeToString(hash.Sum(nil))
		imageHash = strings.ReplaceAll(imageHash, "/", "_")

		filename := "files/" + imageHash
		if _, err = os.Stat(filename); err != nil {
			if err == os.ErrExist {
				return
			}
			err = nil
		}

		if err = ioutil.WriteFile(filename, imageData, 0600); err != nil {
			return
		}

		return
	} else {
		imageHash = v
	}

	return
}
