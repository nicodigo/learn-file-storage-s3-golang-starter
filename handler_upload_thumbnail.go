package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	_ = r.ParseMultipartForm(maxMemory)

	image, imgHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse thumbnail", err)
		return
	}
	defer image.Close()

	mediaType, _, err := mime.ParseMediaType(imgHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "error processing media type", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "must be a png or jpeg image", fmt.Errorf("expected image, but found: %s", mediaType))
		return
	}

	imgMediaExt := strings.Split(mediaType, "/")[1]

	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing request", err)
		return
	}

	if dbVideo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You are not the video owner", err)
		return
	}

	thumbnailFP := filepath.Join(cfg.assetsRoot, videoIDString+"."+imgMediaExt)
	file, err := os.Create(thumbnailFP)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing request", err)
		return
	}

	io.Copy(file, image)

	thumbnailUrl := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, videoIDString, imgMediaExt)

	dbVideo.ThumbnailURL = &thumbnailUrl

	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing request", err)
		return
	}

	respondWithJSON(w, http.StatusOK, dbVideo)
}

