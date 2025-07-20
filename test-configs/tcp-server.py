#!/usr/bin/env python3
import socket
import threading
import time

def handle_client(conn, addr):
    print(f"Connected by {addr}")
    try:
        while True:
            data = conn.recv(1024)
            if not data:
                break
            received = data.decode().strip()
            print(f"Received: {received}")
            if "SELECT 1" in received:
                conn.sendall(b"1\n")
            else:
                conn.sendall(b"OK\n")
    except Exception as e:
        print(f"Error: {e}")
    finally:
        conn.close()

def tcp_server():
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        s.bind(("0.0.0.0", 9090))
        s.listen()
        print("TCP server listening on port 9090")
        while True:
            try:
                conn, addr = s.accept()
                thread = threading.Thread(target=handle_client, args=(conn, addr))
                thread.daemon = True
                thread.start()
            except Exception as e:
                print(f"Server error: {e}")

if __name__ == "__main__":
    tcp_server() 