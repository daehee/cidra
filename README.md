# cidra

Convert IPs to CIDR ranges to expand network attack surface

## Installation

```
go get -u github.com/daehee/cidra
```

Download the latest ip2asn.com database to working directory:
```
wget https://iptoasn.com/data/ip2asn-combined.tsv.gz
```

## Usage

Pipe in list of IP addresses resolved from target subdomains:

```
cat giant-list-of-ips.txt | cidra
```

Add resulting CIDRs to your next `nmap` or `masscan` run.
