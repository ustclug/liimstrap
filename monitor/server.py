#!/usr/bin/env python3

from http.server import BaseHTTPRequestHandler, HTTPServer
from socketserver import ThreadingMixIn
from types import SimpleNamespace
from collections import OrderedDict
import cgi
import datetime
import json
import time

PORT_NUMBER = 3000
TIMEOUT = 120

clients_data = OrderedDict() 

class HTTPHandler(BaseHTTPRequestHandler):

    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-type','text/html')
        self.end_headers()
        self.wfile.write(
            b'<html>'
            b'<head>'
            b'<title>Monitor</title>'
            b'<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@4.4.1/dist/css/bootstrap.min.css" integrity="sha256-L/W5Wfqfa0sdBNIKN9cG6QA5F2qx4qICmU2VgLruv9Y=" crossorigin="anonymous">'
            b'<meta charset="UTF-8">'
            # b'<style>th, td { padding: 2px; }</style>'
            b'</head>'
            b'<body><div class="container p-3">')

        self.wfile.write(b'<table class="table" border="1">')
        self.wfile.write(
            b"<tr>"
            b"<th>Location</th>"
            b"<th>Version</th>"
            b"<th>MAC address</th>"
            b"<th>IP address</th>"
            b"<th>Last reply</th>"
            b"<th>Uptime</th>"
            b"<th>Status</th></tr>")

        for mac, data in clients_data.items():
            now = time.time()
            last_reply = data.get("time", 0.0)
            delta = now - last_reply

            if delta > TIMEOUT:
                status_color = "#FF4136"
                status = "Down"
            else:
                status_color = "#01FF70"
                status = "Ok"


            self.wfile.write(b"<tr>")
            self.wfile.write(
                "<td>{name}</td>"\
                "<td>{version}</td>"\
                "<td>{mac}</td>"\
                "<td>{ip}</td>"\
                "<td>{time}</td>"\
                "<td>{uptime}</td>"\
                "<td style='background-color:{status_color}'>{status}</td>"\
                .format(
                    name = data.get("name"),
                    version = data.get("version"),
                    mac = ":".join([mac[i:i + 2] for i in range(0, len(mac), 2)]),
                    ip = data.get("ip"),
                    time = time.strftime('%Y-%m-%d %H:%M:%S',
                                         time.localtime(last_reply)),
                    uptime = datetime.timedelta(
                                 seconds=float(data.get("uptime", 0.0))),
                    status_color = status_color,
                    status = status)\
                .encode()
            )
            self.wfile.write(b"</tr>")

        self.wfile.write(b"</table></div></body>")

    def do_POST(self):
        form = cgi.FieldStorage(
                    fp=self.rfile, 
                    headers=self.headers,
                    environ={
                        'REQUEST_METHOD':'POST',
                        'CONTENT_TYPE': self.headers['Content-Type'],
                    })

        reply = {}
        reply["ip"] = self.address_string()
        reply["time"] = time.time()

        for k in ["mac", "version", "uptime"]:
            if k in form:
                reply[k] = form[k].value
            else:
                return

        if reply["mac"] in clients_data:
            mac = reply.pop("mac")
            clients_data[mac].update(reply)
        elif "" in clients_data:
            reply.pop("mac")
            clients_data[""].update(reply)

        self.send_response(200)
        self.send_header('Content-type','text/plain')
        self.end_headers()
        self.wfile.write(b"OK")


class ForkingHTTPServer(ThreadingMixIn, HTTPServer):
    def finish_request(self, request, client_address):
        request.settimeout(30)
        # "super" can not be used because BaseServer is not created from object
        HTTPServer.finish_request(self, request, client_address)


def httpd(server_address=('', 3000)):
    try:
        server = ForkingHTTPServer(server_address, HTTPHandler)
        print('Started httpserver on port %d' % PORT_NUMBER)
        server.serve_forever()
    except KeyboardInterrupt:
        print('^C received, shutting down the web server')
        server.socket.close()


if __name__ == "__main__":
    with open("clients.json") as fin:
        clients_config = json.load(fin)

    for item in clients_config["machines"]:
        clients_data[item["mac"]] = {
            "name": item["name"],
        }

    httpd()


