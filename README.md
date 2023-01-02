# Kasa Smart Plug

This is a command-line utility for controlling, and [Prometheus exporter](https://prometheus.io/docs/instrumenting/exporters/) for monitoring, [TP-Link's Kasa Smart Plugs](https://tp-link.com/uk/smarthome).

I test this against my [Kasa KP115](https://www.tp-link.com/uk/home-networking/smart-plug/kp115/).

## Background

I wanted to remote control and monitor the energy usage of a few devices in my house using custom software, so I bought a few of TP-Link's Kasa Smart Plugs as reviews online said they had an open API that exposed lots of data and did not require linking an online account (both of which turned out to be true).

I originally made this around February 2022 as my first actual Go project, however I only recently got around to improving & publishing it here, hence why my [RCON](https://github.com/viral32111/rcon) and [Healthcheck](https://github.com/viral32111/healthcheck) projects were published before this one.

## License

Copyright (C) 2022-2023 [viral32111](https://viral32111.com).

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see https://www.gnu.org/licenses.
