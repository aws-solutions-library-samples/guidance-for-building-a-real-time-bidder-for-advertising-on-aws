import requests
import numpy as np

from numpy import random
x_0 = random.randint(50, size=(1, 17))
# print(x_0)

inference_request = {
    "inputs": [
        {
          "name": "predict-prob",
          "shape": x_0.shape,
          "datatype": "FP32",
          "data": x_0.tolist()
        }
    ]
}

endpoint = "http://localhost:8094/v2/models/ctr-lgbm/versions/v0.1.0/infer"
response = requests.post(endpoint, json=inference_request)

print(response.json())
