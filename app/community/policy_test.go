package community

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func TestValidateDriverRatingBoundsAndUnicodeLength(t *testing.T) {
	for _, score := range []int{1, 5} {
		if err := ValidateDriverRating(score, strings.Repeat("ñ", 1000)); err != nil {
			t.Fatalf("score %d rejected: %v", score, err)
		}
	}
	for _, score := range []int{0, 6, -1} {
		if err := ValidateDriverRating(score, ""); err == nil {
			t.Fatalf("score %d should be rejected", score)
		}
	}
	if err := ValidateDriverRating(5, strings.Repeat("ñ", 1001)); err == nil {
		t.Fatal("comment over 1000 runes should be rejected")
	}
}

func TestValidateProfilePhotoAcceptsOnlyValidSmallJPEGAndPNG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.White)
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, img); err != nil {
		t.Fatal(err)
	}
	if ext, err := ValidateProfilePhoto(pngData.Bytes()); err != nil || ext != ".png" {
		t.Fatalf("PNG rejected: ext=%q err=%v", ext, err)
	}
	var jpegData bytes.Buffer
	if err := jpeg.Encode(&jpegData, img, nil); err != nil {
		t.Fatal(err)
	}
	if ext, err := ValidateProfilePhoto(jpegData.Bytes()); err != nil || ext != ".jpg" {
		t.Fatalf("JPEG rejected: ext=%q err=%v", ext, err)
	}
	if _, err := ValidateProfilePhoto([]byte("not an image")); err == nil {
		t.Fatal("invalid bytes should be rejected")
	}
	if _, err := ValidateProfilePhoto(append([]byte("data:image/png;base64,"), pngData.Bytes()...)); err == nil {
		t.Fatal("mismatched MIME content should be rejected")
	}
}

func TestValidateProfilePhotoRejectsOversizedAndHugeDimensions(t *testing.T) {
	if _, err := ValidateProfilePhoto(bytes.Repeat([]byte{'x'}, MaxProfilePhotoBytes+1)); err == nil {
		t.Fatal("oversized image should be rejected")
	}
	img := image.NewRGBA(image.Rect(0, 0, 4097, 1))
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateProfilePhoto(encoded.Bytes()); err == nil {
		t.Fatal("too-wide image should be rejected")
	}
}

func TestDriverCanSeeLoadEnforcesAudienceAndAssignment(t *testing.T) {
	tests := []struct {
		name, status, audience string
		assigned, member, want bool
	}{
		{"public published", "publicada", "load_board", false, false, true},
		{"private member", "publicada", "red_privada", false, true, true},
		{"private outsider", "publicada", "red_privada", false, false, false},
		{"unknown audience", "publicada", "red_extendida", false, true, false},
		{"cancelled public", "cancelada", "load_board", false, false, false},
		{"assigned historical", "entregada", "red_privada", true, false, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := DriverCanSeeLoad(test.status, test.audience, test.assigned, test.member)
			if got != test.want {
				t.Fatalf("got %t, want %t", got, test.want)
			}
		})
	}
}

func TestValidateAudienceRejectsUnsupportedScope(t *testing.T) {
	for _, value := range []string{"load_board", "red_privada"} {
		if err := ValidateAudience(value); err != nil {
			t.Errorf("%q rejected: %v", value, err)
		}
	}
	for _, value := range []string{"", "red_extendida", "publica", "privada"} {
		if err := ValidateAudience(value); err == nil {
			t.Errorf("%q should be rejected", value)
		}
	}
}
