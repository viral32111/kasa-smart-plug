# Kasa Smart Plug

[![CI](https://github.com/viral32111/kasa-smart-plug/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/viral32111/kasa-smart-plug/actions/workflows/ci.yml)
[![CodeQL](https://github.com/viral32111/kasa-smart-plug/actions/workflows/codeql.yml/badge.svg)](https://github.com/viral32111/kasa-smart-plug/actions/workflows/codeql.yml)
![GitHub tag (with filter)](https://img.shields.io/github/v/tag/viral32111/kasa-smart-plug?label=Latest)
![GitHub repository size](https://img.shields.io/github/repo-size/viral32111/kasa-smart-plug?label=Size)
![GitHub release downloads](https://img.shields.io/github/downloads/viral32111/kasa-smart-plug/total?label=Downloads)
![GitHub commit activity](https://img.shields.io/github/commit-activity/m/viral32111/kasa-smart-plug?label=Commits)

This is a command-line utility for controlling and monitoring (via a [Prometheus exporter](https://prometheus.io/docs/instrumenting/exporters/)) [TP-Link's Kasa Smart Plugs](https://tp-link.com/uk/smarthome).

I test this against my [Kasa KP115](https://www.tp-link.com/uk/home-networking/smart-plug/kp115/).

## 📜 Background

I wanted to remote control and monitor the energy usage of a few devices in my house using custom software, so I bought a few of [TP-Link's Kasa Smart Plugs](https://tp-link.com/uk/smarthome) as online reviews mentioned they had an open API which exposed lots of data and did not require linking an online account, both of which turned out to be true.

I originally made this in February of 2022 as my first Go project. However, it was not published here straight away, hence why my [kasa-smart-plug](https://github.com/viral32111/kasa-smart-plug) and [Healthcheck](https://github.com/viral32111/healthcheck) projects were published before this.

## ⚖️ License

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
