import random
from faker import Faker
from simple_http_client import HttpClient
import json

class User:
    def __init__(self, username, user_id=None):
        self.id = user_id
        self.name = username

    @classmethod
    def from_json(self, jsonStr):
        user_data = json.loads(jsonStr)
        return cls(user_data["name"], user_data("id"))

    def to_json(self):
        user_data = {"name": self.name}
        if self.id is not None:
            user_data["id"] = self.id
        return json.dumps(user_data)



class UserLocation:
    def __init__(self, user_id):
        self.userID = user_id
        self.longitude = random.uniform(-180, 180)
        self.latitude = random.uniform(-90, 90)

    def __str__(self):
        return json.dumps({
            "userID": self.userID,
            "longitude": self.longitude,
            "latitude": self.latitude
        })

    def generate_location_from_center(self, center_longitude, center_latitude, max_distance_miles=8):
        # Convert max_distance_miles to degrees
        max_distance_degrees = max_distance_miles / 69.0

        # Generate random angles
        angle = random.uniform(0, 2 * math.pi)
        distance = random.uniform(0, max_distance_degrees)

        # Convert polar coordinates to Cartesian coordinates
        delta_longitude = distance * math.cos(angle)
        delta_latitude = distance * math.sin(angle)

        # Apply the deltas to the center coordinates
        self.longitude = center_longitude + delta_longitude
        self.latitude = center_latitude + delta_latitude


class UserGenerator:
    def __init__(self):
        self.fake = Faker()

    def generate_username(self):
        # Generate a username using the first name and a random number
        return self.fake.first_name() + str(random.randint(1000, 9999))

    def generate_user(self):
        username = self.generate_username()
        return User(username=username).to_json()

    def generate_users(self, count):
        users = []
        for _ in range(count):
            users.append(self.generate_user())
        return users

    def save_to_file(self, users, output_file):
        with open(output_file, 'w') as file:
            json.dump(users, file)

    def read_from_file(self, output_file):
        users = []
        with open(output_file, 'r') as file:
            user_strings = json.load(file)
            for user_str in user_strings:
                user = User.from_json(json.loads(user_str))
                users.append(user)
        return users


if __name__ == "__main__":
    #generator = UserGenerator()
    #for u in generator.generate_users(10):
    #    print(u)

    generator = UserGenerator()
    generator.save_to_file(generator.generate_users(10), "data/raw_users.json")

    # Read users from file
    #users_from_file = generator.read_from_file("data/raw_users.json")
    #base_url = "http://localhost:8080"
    #http_client = HttpClient(base_url)
    #users_with_id = []
    #for user in users_from_file:
    #    resp = http_client.post("/user/create", data=user.to_json())
    #    users_with_id.append(User.from_json(resp.text)

    #popularUser = users_with_id[0]
    #for user in users_from_file[1:]:
    #    print("making friends with {}: {}", popularUser.to_json(), user.to_json())


