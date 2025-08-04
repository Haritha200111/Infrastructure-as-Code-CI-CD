import mediapipe as mp
import sys
import json
import numpy as np
import cv2
import os
import random

# Initialize MediaPipe face detection
mp_face_detection = mp.solutions.face_detection
mp_drawing = mp.solutions.drawing_utils
face_detection = mp_face_detection.FaceDetection(model_selection=1, min_detection_confidence=0.6)

# Read image from stdin
image_bytes = sys.stdin.buffer.read()
image_name = sys.argv[1]  # Receive image name as command-line argument

# Convert bytes to numpy array
nparr = np.frombuffer(image_bytes, np.uint8)

# Decode image
img = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

# Convert image to RGB (MediaPipe requires RGB input)
img_rgb = cv2.cvtColor(img, cv2.COLOR_BGR2RGB)

# Detect faces
results = face_detection.process(img_rgb)

# Prepare JSON response
face_list = []
if results.detections:
    annotated_img = img.copy()
    for detection in results.detections:
        bboxC = detection.location_data.relative_bounding_box
        ih, iw, _ = img.shape
        bbox = [
            int(bboxC.xmin * iw),
            int(bboxC.ymin * ih),
            int(bboxC.width * iw),
            int(bboxC.height * ih)
        ]
        face_list.append({
            'x': bbox[0],
            'y': bbox[1],
            'width': bbox[2],
            'height': bbox[3]
        })
        mp_drawing.draw_detection(annotated_img, detection)
        directory = 'output_images'
        if not os.path.exists(directory):
            os.makedirs(directory)
        path = directory + '/processed_' + image_name
    cv2.imwrite(path, annotated_img)

response = {
    'success': True,
    'faces': face_list
}

# Output JSON
print(json.dumps(response))
