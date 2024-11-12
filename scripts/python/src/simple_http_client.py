import requests

class HttpClient:
    def __init__(self, base_url):
        self.base_url = base_url

    def _build_url(self, endpoint):
        return f"{self.base_url}/{endpoint.lstrip('/')}"

    def get(self, endpoint, params=None, headers=None):
        url = self._build_url(endpoint)
        response = requests.get(url, params=params, headers=headers)
        return response

    def post(self, endpoint, data=None, json=None, headers=None):
        url = self._build_url(endpoint)
        response = requests.post(url, data=data, json=json, headers=headers)
        return response

#if __name__ == "__main__":
#    # Example usage:
#
#    # GET request
#    get_response = http_client.get("/posts/1")
#    print("GET Response:", get_response.json())
#
#    # POST request
#    post_data = {"title": "foo", "body": "bar", "userId": 1}
#    post_response = http_client.post("/posts", json=post_data)
#    print("POST Response:", post_response.json())
