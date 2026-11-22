package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

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

	imageData, err := io.ReadAll(image)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error reading the thumbnail", err)
		return
	}
	imgMediaType := imgHeader.Header.Get("Content-Type")

	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing request", err)
		return
	}

	if dbVideo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You are not the video owner", err)
		return
	}

	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	thumbnailUrl := fmt.Sprintf("data:%s;base64,%s", imgMediaType, imageBase64)
	dbVideo.ThumbnailURL = &thumbnailUrl

	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing request", err)
		return
	}

	respondWithJSON(w, http.StatusOK, dbVideo)
}
