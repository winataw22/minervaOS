use anyhow::{Context, Result};
use std::path::{Path, PathBuf};
use std::process::{Child, Command};
use walkdir::WalkDir;

pub struct Zfs {
    target: PathBuf,
    child: Child,
}

impl Zfs {
    pub fn mount<S, P>(backend: S, meta: S, target: P) -> Result<Zfs>
    where
        S: AsRef<str>,
        P: Into<PathBuf>,
    {
        // mount the flist
        let target = target.into();
        std::fs::create_dir_all(&target)?;

        let mut child = Command::new("g8ufs")
            .arg("--ro")
            .arg("--cache")
            .arg(backend.as_ref())
            .arg("--meta")
            .arg(meta.as_ref())
            .arg(&target)
            .spawn()?;

        //wait for the mount
        let result = retry::retry(retry::delay::Exponential::from_millis(200).take(10), || {
            let status = Command::new("mountpoint").arg("-q").arg(&target).status()?;
            match status.success() {
                true => Ok(()),
                false => bail!("not a mount point"),
            }
        });

        match result {
            Err(e) => {
                child.kill()?;
                bail!("failed to mount flist: {:?}", e)
            }
            Ok(_) => {}
        }

        Ok(Zfs {
            target: target,
            child: child,
        })
    }

    pub fn copy<P>(&self, target: P) -> Result<()>
    where
        P: AsRef<Path>,
    {
        debug!(
            "copying from {} -to-> {}",
            &self.target.display(),
            target.as_ref().display()
        );

        for entry in WalkDir::new(&self.target) {
            let entry = entry?;
            let src = entry.path();
            let mut dst = PathBuf::new();
            dst.push(&target);
            dst.push(src.strip_prefix(&self.target)?);

            let typ = entry.file_type();

            use std::fs;
            if typ.is_dir() {
                debug!("creating directory {:?}", dst);
                std::fs::create_dir_all(&dst)
                    .with_context(|| format!("failed to create directory: {:?}", &dst))?;
            } else if typ.is_file() {
                let mut tmp = dst.clone();
                let mut tmp_name: std::ffi::OsString = dst.file_name().unwrap().into();
                tmp_name.push(".partial");
                tmp.set_file_name(tmp_name);

                debug!("installing file {:?}", dst);
                fs::copy(&src, &tmp).context("failed to copy file")?;
                fs::rename(&tmp, &dst).context("failed to rename file")?;
            } else if typ.is_symlink() {
                let mut orig = fs::read_link(src)
                    .with_context(|| format!("failed to read link: {:?}", &src))?;

                debug!("installing link {:?} => {:?}", dst, orig);
                orig = if orig.is_relative() {
                    // relative so we can do it directly
                    orig
                } else {
                    // otherwise, we need to prepend the
                    // target directory
                    let mut abs = PathBuf::new();
                    abs.push(&target);
                    abs.push(orig);
                    abs
                };

                match fs::remove_file(&dst) {
                    Ok(_) => {}
                    Err(err) => match err.kind() {
                        std::io::ErrorKind::NotFound => {}
                        _ => bail!("failed to delete file: {:?}", dst),
                    },
                }

                std::os::unix::fs::symlink(orig, dst).context("failed to create symlink")?;
            } else {
                debug!("skipping: ({:?}): {:?}", src, typ)
            }
        }

        Ok(())
    }
}

impl AsRef<Path> for Zfs {
    fn as_ref(&self) -> &Path {
        &self.target
    }
}

impl Drop for Zfs {
    fn drop(&mut self) {
        if let Err(e) = nix::mount::umount2(&self.target, nix::mount::MntFlags::MNT_FORCE) {
            error!("failed to umount flist: {}", e);
        }
        if let Err(e) = self.child.wait() {
            error!("g8ufs wait error: {}", e);
        }
    }
}
