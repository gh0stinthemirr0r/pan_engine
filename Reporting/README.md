# Palo Alto Reports Generator

A Go application for generating reports from Palo Alto Networks firewalls using their REST API. This application provides a web interface for configuring API access and generating various types of reports in both CSV and PDF formats.

## Features

- Web-based interface for easy configuration and report generation
- Secure storage of API credentials
- Multiple report types available through dropdown menu
- Generates reports in both CSV and PDF formats
- Automatic timestamp-based file naming
- Modern Bootstrap-based UI

## Prerequisites

- Go 1.21 or later
- Access to a Palo Alto Networks firewall with API access enabled
- Valid API key from your Palo Alto Networks firewall

## Installation

1. Clone this repository
2. Navigate to the project directory
3. Install dependencies:
   ```bash
   go mod download
   ```

## Usage

1. Start the application:
   ```bash
   go run .
   ```

2. Open your web browser and navigate to:
   ```
   http://localhost:8080
   ```

3. Configure your Palo Alto Networks API settings:
   - Enter your firewall's API URL (e.g., https://your-firewall-ip)
   - Enter your API key

4. Generate reports:
   - Select the desired report type from the dropdown menu
   - Click "Generate Report"
   - Reports will be saved in the `reports` directory

## Report Types

The application supports various report types including:
- System Information
- Interface Information
- Security Rules
- System Resources
- Traffic Logs
- Threat Logs
- URL Filtering Logs
- GlobalProtect Users
- Active Sessions
- System Software Version

## File Formats

Reports are generated in two formats:
- CSV: For data analysis and spreadsheet compatibility
- PDF: For formal documentation and sharing

## Security Notes

- API credentials are stored locally in the config directory
- Use HTTPS for production deployments
- Ensure proper firewall access controls are in place

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details. 