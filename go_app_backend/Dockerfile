# Multi-stage build for Go application
FROM golang:1.22 AS builder

# Install git and other utilities
RUN apt-get update && apt-get install -y git

# Set the working directory
WORKDIR /app

# Copy the Go modules files and download dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the source code from the current directory to the working directory inside the container
COPY . .

# Build the Go app
RUN go build -o app .

# Use Python base image
FROM python:3.12-slim

# Set working directory
WORKDIR /app

# Install necessary system dependencies
RUN apt-get update \
    && apt-get install --assume-yes --no-install-recommends \
        ffmpeg \
        libsm6 \
        libxext6 \
        redis-server \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Mediapipe, OpenCV Python dependencies
RUN pip install --no-cache --upgrade pip setuptools \
    && pip install mediapipe opencv-python-headless

# Copy built Go executable from the previous stage
COPY --from=builder /app/app .

# Copy Python script and other necessary files
COPY face_detection_mediapipe.py .

# Expose ports for the application and Redis server
EXPOSE 8080
EXPOSE 6379

# Command to start the services
CMD ["bash", "-c", "\
    service redis-server start && \
    ./app \
"]

# Optionally, mount a volume for output images (if needed)
VOLUME /app/output_images

# 