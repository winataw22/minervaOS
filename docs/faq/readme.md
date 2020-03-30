# FAQ

This section consolidated all the common question we get about how 0-OS work and how to operate it.

- **Q**: What is the preferred configuration for my raid controller when running 0-OS ?  
  **A**: 0-OS goal is to expose raw capacity. So it is best to always try to give him access to the most raw access to the disks. In case of raid controllers, the best is to try to set it up in [JBOD](https://en.wikipedia.org/wiki/Non-RAID_drive_architectures#JBOD) mode if available.
