````markdown

## Screenshots

![PAN_ENGINE Screenshot 01](https://raw.githubusercontent.com/gh0stinthemirr0r/pan_engine/main/screenshots/PAN_ENGINE_screenshot01.png)  
![PAN_ENGINE Screenshot 02](https://raw.githubusercontent.com/gh0stinthemirr0r/pan_engine/main/screenshots/PAN_ENGINE_screenshot02.png)  
![PAN_ENGINE Screenshot 03](https://raw.githubusercontent.com/gh0stinthemirr0r/pan_engine/main/screenshots/PAN_ENGINE_screenshot03.png)  
![PAN_ENGINE Screenshot 04](https://raw.githubusercontent.com/gh0stinthemirr0r/pan_engine/main/screenshots/PAN_ENGINE_screenshot04.png)  
![PAN_ENGINE Screenshot 05](https://raw.githubusercontent.com/gh0stinthemirr0r/pan_engine/main/screenshots/PAN_ENGINE_screenshot05.png)  
![PAN_ENGINE Screenshot 06](https://raw.githubusercontent.com/gh0stinthemirr0r/pan_engine/main/screenshots/PAN_ENGINE_screenshot06.png)  
![PAN_ENGINE Screenshot 07](https://raw.githubusercontent.com/gh0stinthemirr0r/pan_engine/main/screenshots/PAN_ENGINE_screenshot07.png)  
![PAN_ENGINE Screenshot 08](https://raw.githubusercontent.com/gh0stinthemirr0r/pan_engine/main/screenshots/PAN_ENGINE_screenshot08.png)  
![PAN_ENGINE Screenshot 09](https://raw.githubusercontent.com/gh0stinthemirr0r/pan_engine/main/screenshots/PAN_ENGINE_screenshot09.png)  
![PAN_ENGINE Screenshot 10](https://raw.githubusercontent.com/gh0stinthemirr0r/pan_engine/main/screenshots/PAN_ENGINE_screenshot10.png)  


# PAN_ENGINE – Palo Alto Networks API Interface

PAN_ENGINE is a standalone Go application that provides a powerful interface to Palo Alto Networks firewalls and Panorama.  
It leverages the official PAN-OS APIs to query, manage, and report on firewall configurations in a fast, portable binary.  

---

## Features

- **API Management**  
  - Configure API URLs, tokens, and connection profiles.  
  - Test connectivity before executing queries.  
  - Save multiple environments (lab, production, multi-tenant).  

- **Firewall & Panorama Queries**  
  - Security, NAT, and PBF rule listings.  
  - Address, service, and application objects.  
  - Rule hit count insights (future release).  

- **System & Health Monitoring**  
  - Device info and HA status.  
  - Session table metrics and active connections.  
  - License, dynamic updates, and threat feed status.  

- **Reporting Engine**  
  - Export to **CSV, JSON, or PDF**.  
  - Generate compliance-ready audit reports.  
  - Integrate into SIEM/SOAR or monitoring pipelines.  

- **Cross-Platform Binary**  
  - Built in Go — compile once, run anywhere.  
  - CLI-first design with optional TUI/GUI roadmap.  

---

## Use Cases

- Automate auditing of firewall rules.  
- Retrieve and export Palo Alto data at scale.  
- Continuous monitoring of health and licensing.  
- Embed in automation pipelines or CI/CD workflows.  

---

## Installation

### Prerequisites
- Go 1.22+  
- Palo Alto firewall or Panorama with API key access  

### Clone and Build
```bash
git clone https://github.com/<your-org>/PAN_ENGINE.git
cd PAN_ENGINE
go build -o pan_engine ./cmd/pan_engine
````

### Run

```bash
./pan_engine --api https://<firewall-ip> --token <api-key> --list-policies
```

---

## Example Usage

List all security policies:

```bash
./pan_engine --api https://fw.example.com --token <api-key> --list-policies
```

Export objects to JSON:

```bash
./pan_engine --api https://fw.example.com --token <api-key> --export objects.json
```

Check HA status:

```bash
./pan_engine --api https://fw.example.com --token <api-key> --ha-status
```

---

## Roadmap

* [ ] Panorama multi-device support
* [ ] Rule hit counter visualization
* [ ] NetBox asset integration
* [ ] REST API mode
* [ ] TUI/GUI dashboard

---

## Screenshots

*(Insert screenshots of CLI output, example reports, or dashboards if available)*

---

## License

MIT License – free to use and modify. See [LICENSE](LICENSE) for details.

```

---


