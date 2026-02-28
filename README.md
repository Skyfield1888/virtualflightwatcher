# Virtual Flight Watcher

A Discord bot that lets you look up live flight and ATC information from the **VATSIM** network directly in your Discord server.
IVAO compatibility is expected, but not in the near future 

---

## Features

- **Pilot lookup** — Get real-time position, flight plan, altitude, speed, and more from the network
- **Controller lookup** — Find active ATC stations with frequency and ATIS
- **ATIS lookup** — Get the current ATIS of any active station
- **Prefile lookup** — View filed flight plans before departure
- **Facility / Rating info** — Look up VATSIM facilities, ATC ratings, pilot ratings, and military ratings
- **Live data** — VATSIM data is refreshed every 15 seconds in the background with zero impact on response time

---

## Commands

| Command | Description | Options |
|---|---|---|
| `/pilot` | Get live pilot info | `network`, `cid_or_callsign` |
| `/controller` | Get live controller info | `network`, `cid_or_callsign` |
| `/atis` | Get the ATIS of a station | `network`, `cid_or_callsign` |
| `/prefile` | Get a filed flight plan | `network`, `cid_or_callsign` |
| `/facility` | Get a VATSIM facility | `id_or_name` (e.g. `5` or `APP`) |
| `/rating` | Get an ATC rating | `id_or_name` (e.g. `3` or `S2`) |
| `/pilotrating` | Get a pilot rating | `id_or_name` |
| `/militaryrating` | Get a military rating | `id_or_name` |

> **Tip:** For `/atis`, you don't need to include `_ATIS` — just type the station name (e.g. `LFPG` or `LFPG_APP`) and the bot handles it automatically.

---

## Getting Started

### Prerequisites

- [Go](https://go.dev/) 1.21+
- A Discord bot token ([create one here](https://discord.com/developers/applications))
- A VATSIM account (optional, for testing)

### Installation

```bash
git clone https://github.com/Skyfield1888/Vatsim.git
cd Vatsim
```

### Configuration

Add in the `.env` file at the root of the project:

```env
DISCORD_TOKEN=your_discord_bot_token_here
```

> Never commit your `.env` file.

### Running

```bash
go run main.go
```

---

## Project Structure

```
Vatsim/
├── main.go                  
├── api/
│   └── vatsim/
│       └── vatsim.go        
├── commands/
│   └── commands.go          
├── .env                     
└── go.mod
```

---

## How It Works

On startup, the bot fetches the full VATSIM data feed (`https://data.vatsim.net/v3/vatsim-data.json`) and indexes every pilot, controller, ATIS, prefile, facility, and rating into in-memory hash maps. A background goroutine refreshes the data every 15 seconds. All Discord command lookups are instant O(1) map reads — no API call is made per command.

---

## Dependencies

| Package | Purpose |
|---|---|
| [`bwmarrin/discordgo`](https://github.com/bwmarrin/discordgo) | Discord API client |
| [`joho/godotenv`](https://github.com/joho/godotenv) | `.env` file loading |

---

## Roadmap

- [ ] IVAO network support
- [ ] Flight tracking / route map
- [ ] Event notifications
- [ ] Prefile Notifications on new Prefiles for a set Facility for controlers
