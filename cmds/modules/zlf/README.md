# DEPRECATED

# zlf
Zero-OS Logs Forwarder reads logs from local unix socket and forward them to a remote redis server

# Usage
Usage of ./zlf:
  -channel string
    	redis logs channel name (default "zinit-logs")
  -host string
    	redis host (default "localhost")
  -logs string
    	zinit unix socket (default "/var/run/log.sock")
  -port int
    	redis port (default 6379)
