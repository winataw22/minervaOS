use std::fmt::Display;

use super::kparams;
use anyhow::Result;
use clap::{App, Arg};

#[derive(Debug)]
pub enum RunMode {
    Prod,
    Test,
    Dev,
    QA,
}

impl Display for RunMode {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let d = match self {
            Self::Prod => "production",
            Self::QA => "qa",
            Self::Test => "testing",
            Self::Dev => "development",
        };

        f.write_str(d)
    }
}

#[derive(Debug)]
pub enum Version {
    V3,
    V4,
}

fn runmode() -> Result<RunMode> {
    let params = kparams::params()?;
    let mode = match params.get("runmode") {
        Some(mode) => match mode {
            Some(mode) => match mode.as_ref() {
                "prod" | "production" => RunMode::Prod,
                "dev" | "development" => RunMode::Dev,
                "test" | "testing" => RunMode::Test,
                "qa" => RunMode::QA,
                m => {
                    bail!("unknown runmode: {}", m);
                }
            },
            None => {
                //that's an error because runmode was
                //provided as a kernel argumet but with no
                //value
                bail!("missing runmode value");
            }
        },
        // runmode was not provided in cmdline
        // so default is prod
        None => RunMode::Prod,
    };

    Ok(mode)
}

fn version() -> Result<Version> {
    let params = kparams::params()?;
    let ver = match params.get("version") {
        Some(input) => match input {
            Some(input) => match input.as_ref() {
                "v3" => Version::V3,
                "v4" => Version::V4,
                m => {
                    bail!("unknown version: {}", m);
                }
            },
            None => Version::V3,
        },
        // version was not provided in cmdline
        // so default is v3
        None => Version::V3,
    };

    Ok(ver)
}

pub struct Config {
    pub stage: u32,
    pub debug: bool,
    pub runmode: RunMode,
    pub version: Version,
}

impl Config {
    pub fn current() -> Result<Config> {
        let matches = App::new("bootstrap")
            .author("Muhamad Azmy <muhamad.azmy@gmail.com>")
            .about("bootstraps zos from minimal image")
            .arg(
                Arg::with_name("stage")
                    .short("s")
                    .value_name("STAGE")
                    .takes_value(true)
                    .required(false)
                    .default_value("1")
                    .help("specify the bootstrap starting stage"),
            )
            .arg(
                Arg::with_name("debug")
                    .short("d")
                    .takes_value(false)
                    .help("run in debug mode, will use the bootstrap:development.flist"),
            )
            .get_matches();

        let stage: u32 = match matches.value_of("stage").unwrap().parse() {
            Ok(stage) => stage,
            Err(err) => {
                bail!("invalid stage format expecting a positive integer: {}", err);
            }
        };

        if stage == 0 {
            bail!("invalid stage value 0, stages starting from 1");
        }
        let mut ver = Version::V3;
        if stage == 2 {
            ver = version()?
        }
        Ok(Config {
            stage,
            debug: matches.occurrences_of("debug") > 0,
            runmode: runmode()?,
            version: ver,
        })
    }
}
