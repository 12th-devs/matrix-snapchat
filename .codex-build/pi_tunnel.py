import socket, threading, select, paramiko, sys, time
HOST='192.168.12.53'; USER='cole'; PASSWORD='1225'
FORWARDS=[(6080,'127.0.0.1',6080),(3101,'127.0.0.1',3101),(29344,'127.0.0.1',29344)]

def handler(client_sock, chan):
    try:
        while True:
            r,_,_=select.select([client_sock, chan], [], [], 60)
            if client_sock in r:
                data=client_sock.recv(16384)
                if not data: break
                chan.sendall(data)
            if chan in r:
                data=chan.recv(16384)
                if not data: break
                client_sock.sendall(data)
    finally:
        try: chan.close()
        except Exception: pass
        try: client_sock.close()
        except Exception: pass

def listener(local_port, remote_host, remote_port):
    sock=socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    sock.bind(('127.0.0.1', local_port)); sock.listen(20)
    while True:
        client_sock, addr=sock.accept()
        try:
            chan=transport.open_channel('direct-tcpip', (remote_host, remote_port), addr)
        except Exception:
            client_sock.close(); continue
        threading.Thread(target=handler, args=(client_sock, chan), daemon=True).start()

ssh=paramiko.SSHClient(); ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
ssh.connect(HOST, username=USER, password=PASSWORD, timeout=15, banner_timeout=15, auth_timeout=15)
transport=ssh.get_transport()
for f in FORWARDS:
    threading.Thread(target=listener, args=f, daemon=True).start()
print('paramiko tunnels listening on 127.0.0.1:6080, 3101, 29344', flush=True)
while True:
    time.sleep(3600)
