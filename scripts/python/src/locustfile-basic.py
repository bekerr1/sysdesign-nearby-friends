from locust import HttpUser, TaskSet, task, between, events
from faker import Faker
from websocket import create_connection
from pprint import pprint

import time
import random
import json

from uuid import uuid4

users = []
userFriendCount = {}

class User:
    def __init__(self):
        self.name = Faker().first_name() + str(random.randint(1000, 9999))
        self.id = None
        #self.id = random.randint(1000, 9999)


def get_random_user():
    if len(users) < 1:
        return None
    randUserIdx = random.randint(0, len(users)-1)
    return users[randUserIdx]

class Basic(TaskSet):
    userObj = None

    def on_start(self):
        self.userObj = User()
        print("On start: posting json '{}'".format(self.userObj.__dict__))
        with self.client.post("/user/register", json=self.userObj.__dict__, catch_response=True) as resp:
            pprint(vars(resp))
            if resp.status_code == 201:
                user = resp.json()
                print("adding user {} to users collection".format(user))
                users.append(user)
                userFriendCount[user["id"]] = 0
            else:
                print("got non-success status code from response: {}".format(resp.status_code))
                return

    @task
    def create_friend(self):
        randUser = get_random_user()
        if randUser is None or len(users) < 2:
            print("not enough users created yet")
            return
        print("selected random user to start friendship with: {}".format(randUser))
        with self.client.get("/user/{}/possible-friends".format(randUser["id"]), catch_response=True) as resp:
            pprint(vars(resp))
            if resp.status_code == 200:
                possibleFriends = resp.json()
            elif resp.status_code == 409:
                raise StopLocust()
            else:
                print("got non-success unhandled status code from response: {}".format(resp.status_code))
                return
        if possibleFriends is None:
            print("No possible users to connect with")
            return
        print("possible friends to connect with: {}".format(possibleFriends))
        randFriendIdx = random.randint(0, len(possibleFriends)-1)
        randFriend = possibleFriends[randFriendIdx]
        print("randomly selected friend: {}".format(randFriend))

        fr = {"user": randUser, "friend": randFriend}
        print("posting friend request: {}".format(fr))
        with self.client.post("/user/friendship", json=fr, catch_response=True) as resp:
            pprint(vars(resp))
            if resp.status_code == 201:
                possibleFriends = resp.json()
                userFriendCount[randUser["id"]] += 1
            else:
                print("got non-success status code from response: {}".format(resp.status_code))
                return


class WS(TaskSet):
    def on_start(self):
        ws = create_connection('ws://0.0.0.0:8080')
        self.ws = ws

        def _receive():
            while True:
                res = ws.recv()
                data = json.loads(res)
                end_at = time.time()
                response_time = int((end_at - data['start_at']) * 1000000)
                events.request_success.fire(
                    request_type='WebSocket Recv',
                    name='test/ws/echo',
                    response_time=response_time,
                    response_length=len(res),
                )
        gevent.spawn(_receive)

    def on_quit(self):
        self.ws.close()

    @task
    def update_location(self):
        start_at = time.time()
        body = json.dumps({'message': 'hello, world', 'user_id': self.user_id, 'start_at': start_at})
        self.ws.send(body)
        events.request_success.fire(
            request_type='WebSocket Sent',
            name='test/ws/echo',
            response_time=int((time.time() - start_at) * 1000000),
            response_length=len(body),
        )

class NearbyFriendsUser(HttpUser):
    tasks = [Basic]
    wait_time = between(5, 15)

#user1 = User()
#user2 = User()
#user3 = User()
#
#users[user1.id] = user1
#users[user2.id] = user2
#users[user3.id] = user3
#
#for k, v in users.items():
#    print(v.__dict__)
