# PAN_ENGINE: Palo Alto Networks API Interface

## About

PAN_ENGINE is a desktop application that provides a complete graphical interface for interacting with Palo Alto Networks firewalls through their REST and XML APIs. The application allows users to generate reports, export data, search across configurations, and manage firewall settings all from a user-friendly GUI.

## Features

- **Complete API Access**: Connect to any Palo Alto Networks device via its API
- **Secure Credential Storage**: API keys are encrypted at rest
- **Comprehensive Reporting**: Generate reports for all aspects of your Palo Alto environment
- **Multiple Export Formats**: Export as CSV or PDF with professional formatting
- **Batch Operations**: Run multiple reports simultaneously with parallel processing
- **Search Functionality**: Search across all report data with powerful filtering
- **Data Visualization**: View your network data in intuitive tables and charts
- **Report Management**: Save, organize, and reuse reports

## Getting Started

### Prerequisites

- A Palo Alto Networks firewall or Panorama with API access enabled
- API key with appropriate permissions
- Windows, macOS, or Linux operating system

### Installation

1. Download the latest release for your platform from the releases page
2. Install the application following the platform-specific instructions
3. Launch the application
4. Configure your API connection in the settings

## Development

This project is built with:
- [Wails](https://wails.io/) for the cross-platform framework
- Go backend for API communication and data processing
- Svelte frontend for the user interface

### Live Development

To run in live development mode:

```bash
wails dev
```

This will run a Vite development server with hot reload for frontend changes. 

For browser-based development with access to Go methods, a dev server runs on http://localhost:34115.

### Building

To build a redistributable package:

```bash
wails build
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- The Palo Alto Networks API documentation
- The Wails framework team
- The Go and Svelte communities

---

*This software is not affiliated with Palo Alto Networks, Inc.*
