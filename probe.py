"""Send three IPv4 UDP probes with the same ID to your deployment."""

import argparse
import secrets
import socket
import time


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("host", help="public UDP hostname or IPv4 address")
    parser.add_argument("port", type=int, help="public UDP port")
    parser.add_argument("--local-port", type=int, default=0, help="local source port, 0 chooses a free port")
    args = parser.parse_args()
    if not 1 <= args.port <= 65535 or not 0 <= args.local_port <= 65535:
        parser.error("port numbers must be in range (destination 1-65535, source 0-65535)")

    probe_id = secrets.token_hex(16)
    payload = f"UDPDISC1 {probe_id}".encode("ascii")
    try:
        with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as udp:
            target = (socket.gethostbyname(args.host), args.port)
            udp.bind(("0.0.0.0", args.local_port))
            print(f"probe_id={probe_id} local_port={udp.getsockname()[1]} target={target[0]}:{target[1]}", flush=True)
            for attempt in range(3):
                if attempt:
                    time.sleep(0.2)
                udp.sendto(payload, target)
    except OSError as error:
        raise SystemExit(f"UDP send failed: {error}") from error


if __name__ == "__main__":
    main()
