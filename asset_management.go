package tupa

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func UploadFile(tc *TupaContext, filePrefix, destFolder, formFileKey string) (multipart.FileHeader, error) {
	tc.Request().ParseMultipartForm(10 << 20)

	file, fileHeader, err := tc.Request().FormFile(formFileKey)
	if err != nil {
		fmt.Println("Erro ao retornar o arquivo")
		fmt.Println(err)
		return multipart.FileHeader{}, err
	}

	randStr, err := GenerateRandomStringHelper(6)
	if err != nil {
		return multipart.FileHeader{}, err
	}
	fileHeader.Filename = filePrefix + "_" + randStr + fileHeader.Filename

	defer file.Close()
	// fmt.Printf("Uploaded File: %+v\n", fileHeader.Filename)
	// fmt.Printf("File Size: %+v\n", fileHeader.Size)
	// fmt.Printf("MIME Header: %+v\n", fileHeader.Header)

	destPath := filepath.Join(destFolder, fileHeader.Filename)
	destFile, err := os.Create(destPath)
	if err != nil {
		return multipart.FileHeader{}, err
	}

	defer destFile.Close()
	if err != nil {
		return multipart.FileHeader{}, WriteJSONHelper(*tc.Response(), http.StatusInternalServerError, err.Error())
	}

	// copia o arquivo do upload para o arquivo criado no SO
	if _, err := io.Copy(destFile, file); err != nil {
		return multipart.FileHeader{}, WriteJSONHelper(*tc.Response(), http.StatusInternalServerError, err.Error())
	}

	fmt.Fprint(*tc.Response(), "Arquivo salvo com sucesso\n")
	return *fileHeader, nil
}
