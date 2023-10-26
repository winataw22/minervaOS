use anyhow::Result;
use reqwest::{blocking::get, StatusCode};
use serde::{Deserialize, Serialize};
use serde_json;
use std::fs::{write, OpenOptions};
use std::io::copy;
use std::path::Path;

const HUB: &str = "https://hub.grid.tf";

pub struct Repo {
    name: String,
}

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq)]
pub enum Kind {
    #[serde(rename = "regular")]
    Regular,
    #[serde(rename = "symlink")]
    Symlink,
    #[serde(rename = "tag")]
    Tag,
    #[serde(rename = "taglink")]
    TagLink,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Flist {
    #[serde(rename = "type")]
    pub kind: Kind,
    pub updated: u64,
    #[serde(default)]
    pub md5: String,
    pub name: String,
    pub target: Option<String>,
    #[serde(skip)]
    pub url: String,
}

impl Repo {
    pub fn new<T>(name: T) -> Repo
    where
        T: AsRef<str>,
    {
        Repo {
            name: String::from(name.as_ref()),
        }
    }

    pub fn list(&self) -> Result<Vec<Flist>> {
        let url = format!("{}/api/flist/{}", HUB, self.name,);

        let response = get(&url)?;
        let mut info: Vec<Flist> = match response.status() {
            StatusCode::OK => response.json()?,
            s => bail!("failed to get flist info: {}", s),
        };
        for flist in info.iter_mut() {
            flist.url = format!("{}/{}/{}", HUB, self.name, flist.name);
        }

        Ok(info)
    }

    pub fn list_tag<S: AsRef<str>>(&self, tag: S) -> Result<Option<Vec<Flist>>> {
        let tag = tag.as_ref();

        let url = format!("{}/api/flist/{}/tags/{}", HUB, self.name, tag);
        let response = get(&url)?;
        let mut info: Vec<Flist> = match response.status() {
            StatusCode::OK => response.json()?,
            StatusCode::NOT_FOUND => return Ok(None),
            s => bail!("failed to get flist info: {}", s),
        };

        // when listing tags. The flists inside have target as full name
        // so we need
        for flist in info.iter_mut() {
            if flist.kind == Kind::Regular {
                // this is impossible because tags can only have symlinks
                continue;
            }

            let target = match &flist.target {
                None => {
                    // this is also not possible may be we should return an error
                    // but we can just skip for now
                    continue;
                }
                Some(target) => target,
            };

            flist.url = format!("{}/{}", HUB, target);
        }

        Ok(Some(info))
    }

    /// gets flist information from name. the flist must be of type flist or symlink
    pub fn get<T>(&self, flist: T) -> Result<Flist>
    where
        T: AsRef<str>,
    {
        let url = format!("{}/api/flist/{}/{}/light", HUB, self.name, flist.as_ref());

        let response = get(&url)?;
        let mut info: Flist = match response.status() {
            StatusCode::OK => response.json()?,
            s => bail!("failed to get flist info: {}", s),
        };
        info.url = format!("{}/{}/{}", HUB, self.name, flist.as_ref());
        Ok(info)
    }
}

impl Flist {
    /// tag_link return the target repo and tag name for
    /// a taglink flist. Panic if the flist is not of type TagLink
    pub fn tag_link(self) -> (String, String) {
        if self.kind != Kind::TagLink {
            panic!("invalid flist type must be a taglink");
        }

        let target = self.target.unwrap();
        let parts: Vec<&str> = target.split('/').collect();
        assert_eq!(parts.len(), 3, "link must be 3 parts");
        assert_eq!(parts[1], "tags");

        (parts[0].to_owned(), parts[2].to_owned())
    }

    /// download the flist to `out`, panics if flist kind is
    /// not regular or symlink
    pub fn download<T>(&self, out: T) -> Result<()>
    where
        T: AsRef<Path>,
    {
        match self.kind {
            Kind::Regular | Kind::Symlink => {}
            _ => {
                panic!("invalid flist type, must be regular or symlink")
            }
        }

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
        assert_eq!(flist.kind, Kind::Regular);
        assert_eq!(flist.url, "https://hub.grid.tf/azmy/test.flist");

        Ok(())
    }

    #[test]
    fn test_list_tag() -> Result<()> {
        let repo = Repo::new(String::from("tf-autobuilder"));

        let list = repo.list_tag("3b51aa5")?;

        assert!(list.is_some());
        let list = list.unwrap();

        assert_ne!(list.len(), 0);

        assert!(repo.list_tag("wrong")?.is_none());

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
