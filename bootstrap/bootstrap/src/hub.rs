use anyhow::Result;
use reqwest::{blocking::get, StatusCode};
use serde::{Deserialize, Serialize};
use serde_json;
use std::fs::{write, OpenOptions};
use std::io::copy;
use std::path::Path;

const HUB: &str = "https://hub.grid.tf";

pub struct Repo {
    base: String,
    name: String,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Flist {
    #[serde(rename = "type")]
    pub kind: String,
    pub updated: u64,
    #[serde(default)]
    pub md5: String,
    pub name: String,
    #[serde(default)]
    pub target: String,

    #[serde(skip)]
    pub url: String,
}

impl Repo {
    pub fn new<T>(name: T) -> Repo
    where
        T: AsRef<str>,
    {
        Repo {
            base: String::from(HUB),
            name: String::from(name.as_ref()),
        }
    }

    pub fn list(&self) -> Result<Vec<Flist>> {
        let url = format!("{}/api/flist/{}", self.base, self.name,);

        let response = get(&url)?;
        let mut info: Vec<Flist> = match response.status() {
            StatusCode::OK => response.json()?,
            s => bail!("failed to get flist info: {}", s),
        };
        for flist in info.iter_mut() {
            flist.url = format!("{}/{}/{}", self.base, self.name, flist.name);
        }

        Ok(info)
    }

    pub fn get<T>(&self, flist: T) -> Result<Flist>
    where
        T: AsRef<str>,
    {
        let url = format!(
            "{}/api/flist/{}/{}/light",
            self.base,
            self.name,
            flist.as_ref()
        );

        let response = get(&url)?;
        let mut info: Flist = match response.status() {
            StatusCode::OK => response.json()?,
            s => bail!("failed to get flist info: {}", s),
        };
        info.url = format!("{}/{}/{}", self.base, self.name, flist.as_ref());
        Ok(info)
    }
}

impl Flist {
    /// download the flist to `out`
    pub fn download<T>(&self, out: T) -> Result<()>
    where
        T: AsRef<Path>,
    {
        let mut file = OpenOptions::new().write(true).create(true).open(out)?;

        let mut response = get(&self.url)?;
        if !response.status().is_success() {
            bail!("failed to download flist: {}", response.status());
        }

        copy(&mut response, &mut file)?;
        Ok(())
    }

    /// write the flist info (in json format) to out
    pub fn write<T>(&self, out: T) -> Result<()>
    where
        T: AsRef<Path>,
    {
        write(out, serde_json::to_vec(self)?)?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn test_get_flist() -> Result<()> {
        let repo = Repo::new(String::from("azmy"));
        let flist = repo.get("test.flist")?;
        assert_eq!(flist.name, "test.flist");
        assert_eq!(flist.kind, "regular");
        assert_eq!(flist.url, "https://hub.grid.tf/azmy/test.flist");

        Ok(())
    }

    #[test]
    fn test_download_flist() -> Result<()> {
        let repo = Repo::new(String::from("azmy"));
        let flist = repo.get("test.flist")?;
        let temp = "/tmp/hub-download-test.flist";
        flist.download(temp)?;

        // comput hash sum of the downloaded file
        // and compare it with flist hash info
        use std::process::Command;
        let output = Command::new("md5sum").arg(temp).output()?;
        let output = String::from_utf8(output.stdout)?;
        let line: Vec<&str> = output.split_whitespace().collect();

        assert_eq!(line[0], &flist.md5);
        let _ = std::fs::remove_file(temp);
        Ok(())
    }

    #[test]
    fn test_list_repo() -> Result<()> {
        let repo = Repo::new(String::from("azmy"));
        let lists = repo.list()?;

        let mut found: Option<&Flist> = None;
        for flist in lists.iter() {
            if flist.name == "test.flist" {
                found = Some(flist);
                break;
            }
        }

        let found = found.unwrap();
        assert_eq!(found.name, "test.flist");
        assert_eq!(found.url, "https://hub.grid.tf/azmy/test.flist");

        Ok(())
    }
}
