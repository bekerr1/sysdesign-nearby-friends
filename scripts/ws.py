import websocket

def on_message(ws, message):
    print("Received message:", message)

def on_error(ws, error):
    print("Error:", error)

def on_close(ws):
    print("Connection closed")

def on_open(ws):
    # Send a message when the connection is open
    ws.send("Hello, Server!")

# WebSocket URL
ws_url = "ws://localhost:8080/user/location"

# Establish WebSocket connection
ws = websocket.WebSocketApp(ws_url,
                            on_message=on_message,
                            on_error=on_error,
                            on_close=on_close)
ws.on_open = on_open

# Start WebSocket communication
ws.run_forever()
