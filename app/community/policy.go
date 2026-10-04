package community

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"unicode/utf8"
)

const MaxProfilePhotoBytes = 5 << 20

func ValidateDriverRating(score int, comment string) error {
	if score < 1 || score > 5 {
		return errors.New("el puntaje debe estar entre 1 y 5")
	}
	if utf8.RuneCountInString(comment) > 1000 {
		return errors.New("el comentario no puede superar 1000 caracteres")
	}
	return nil
}

func ValidateProfilePhoto(data []byte) (string, error) {
	if len(data) == 0 || len(data) > MaxProfilePhotoBytes {
		return "", errors.New("la imagen está vacía o supera 5 MB")
	}
	probe := data
	if len(probe) > 512 {
		probe = probe[:512]
	}
	ext := ""
	switch http.DetectContentType(probe) {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	default:
		return "", errors.New("solo se admiten imágenes JPEG o PNG")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 4096 || cfg.Height > 4096 || int64(cfg.Width)*int64(cfg.Height) > 16_000_000 {
		return "", errors.New("la imagen no es válida o supera 16 megapíxeles")
	}
	return ext, nil
}

func DriverCanSeeLoad(status, audience string, assignedToDriver, memberOfCompany bool) bool {
	if assignedToDriver {
		return true
	}
	if status != "publicada" {
		return false
	}
	switch audience {
	case "load_board":
		return true
	case "red_privada":
		return memberOfCompany
	default:
		return false
	}
}

func ValidateAudience(audience string) error {
	if audience != "load_board" && audience != "red_privada" {
		return fmt.Errorf("audiencia inválida")
	}
	return nil
}
