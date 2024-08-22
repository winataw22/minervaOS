use anyhow::Error;
use shlex::split;
use std::collections::HashMap;
use std::fs;

const PARAM_FILE: &str = "/proc/cmdline";
type Params = HashMap<String, Option<String>>;

pub fn params() -> Result<Params, Error> {
    let bytes = fs::read(PARAM_FILE)?;
    parse(&bytes)
}

fn parse(bytes: &[u8]) -> Result<Params, Error> {
    let args = match split(std::str::from_utf8(bytes)?) {
        Some(args) => args,
        None => bail!("failed to parse kernel params"),
    };

    let mut map = Params::new();

    for arg in args {
        let parts: Vec<&str> = arg.splitn(2, '=').collect();
        let key = String::from(parts[0]);
        let value = match parts.len() {
            1 => None,
            _ => Some(String::from(parts[1])),
        };

        map.insert(key, value);
    }

    Ok(map)
}

#[cfg(test)]
mod tests {
    // Note this useful idiom: importing names from outer (for mod tests) scope.
    use super::*;

    #[test]
    fn test_parse() -> Result<(), Error> {
        let input: &str = "initrd=initramfs-linux.img version=v3 root=UUID=10f9e7bb-ba63-4fbd-a95e-c78b5496cfbe rootflags=subvol=root rw b43.allhwsupport=1";
        let result = parse(input.as_bytes())?;
        assert_eq!(result.len(), 6);
        assert_eq!(result["rw"], None);
        assert_eq!(result["version"], Some(String::from("v3")));
        assert_eq!(result["rootflags"], Some(String::from("subvol=root")));
        Ok(())
    }
}
