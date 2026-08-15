//! Offline disposable email domain detection backed by BillionVerify's
//! continuously updated embedded domain list.
//!
//! ```
//! assert!(disposable::is_domain("mailinator.com"));
//! assert!(disposable::is_email("user@mailinator.com"));
//! assert!(!disposable::is_domain("example.com"));
//! ```

#![forbid(unsafe_code)]
#![warn(missing_docs)]

use std::collections::HashSet;
use std::sync::OnceLock;

const EMBEDDED_DOMAINS: &str = include_str!("../../data/domains.txt");
const EMBEDDED_WILDCARDS: &str = include_str!("../../data/wildcards.txt");
const EMBEDDED_EXCEPTIONS: &str = include_str!("../../data/exceptions.txt");

static EMBEDDED: OnceLock<DisposableDomains> = OnceLock::new();

/// An immutable disposable-domain detector built from exact, wildcard, and
/// exception lists.
///
/// Construct a custom detector with [`DisposableDomains::from_lists`]. The
/// crate-level lookup functions use a shared detector built from the embedded
/// repository data.
pub struct DisposableDomains {
    domains: HashSet<String>,
    wildcard_suffixes: Vec<String>,
    exceptions: HashSet<String>,
}

impl DisposableDomains {
    fn embedded() -> Self {
        Self::from_lists(EMBEDDED_DOMAINS, EMBEDDED_WILDCARDS, EMBEDDED_EXCEPTIONS)
    }

    /// Builds a detector from newline-delimited lists.
    ///
    /// Blank lines and lines beginning with `#` are ignored. Wildcard entries
    /// may include the `*.` prefix. Invalid hostnames are silently discarded,
    /// matching the embedded list loader's behavior.
    pub fn from_lists(domains: &str, wildcards: &str, exceptions: &str) -> Self {
        let wildcard_suffixes = parse_list(wildcards, true).into_iter().collect();

        Self {
            domains: parse_list(domains, false),
            wildcard_suffixes,
            exceptions: parse_list(exceptions, false),
        }
    }

    /// Reports whether `domain` matches this detector's data.
    pub fn is_domain(&self, domain: &str) -> bool {
        let Some(domain) = normalize(domain) else {
            return false;
        };

        if self.exceptions.contains(&domain) {
            return false;
        }
        if self.domains.contains(&domain) {
            return true;
        }

        self.wildcard_suffixes.iter().any(|suffix| {
            domain == *suffix
                || domain
                    .strip_suffix(suffix)
                    .is_some_and(|prefix| prefix.ends_with('.'))
        })
    }

    /// Extracts the domain portion of `email` and checks this detector's data.
    pub fn is_email(&self, email: &str) -> bool {
        let Some(separator) = email.rfind('@') else {
            return false;
        };
        if separator == 0 || separator == email.len() - 1 {
            return false;
        }

        self.is_domain(&email[separator + 1..])
    }

    /// Returns the number of distinct exact-match domains in this detector.
    pub fn count(&self) -> usize {
        self.domains.len()
    }
}

fn embedded() -> &'static DisposableDomains {
    EMBEDDED.get_or_init(DisposableDomains::embedded)
}

/// Reports whether `domain` is in the embedded disposable domain list.
///
/// Input is case-insensitive, surrounding whitespace is ignored, and Unicode
/// domain names are converted to their ASCII IDNA representation before
/// lookup. Malformed domains return `false`.
pub fn is_domain(domain: &str) -> bool {
    embedded().is_domain(domain)
}

/// Extracts the domain portion of `email` and checks the embedded list.
///
/// Inputs without a non-empty local part and domain return `false`. This is a
/// convenience lookup, not a complete email address syntax validator.
pub fn is_email(email: &str) -> bool {
    embedded().is_email(email)
}

/// Returns the number of distinct exact-match domains in the embedded list.
pub fn count() -> usize {
    embedded().count()
}

fn parse_list(data: &str, strip_wildcard_prefix: bool) -> HashSet<String> {
    data.lines()
        .map(str::trim)
        .filter(|line| !line.is_empty() && !line.starts_with('#'))
        .map(|line| {
            if strip_wildcard_prefix {
                line.strip_prefix("*.").unwrap_or(line)
            } else {
                line
            }
        })
        .filter_map(normalize)
        .collect()
}

fn normalize(domain: &str) -> Option<String> {
    let domain = domain.trim().to_lowercase();
    if domain.is_empty() {
        return None;
    }

    let ascii = idna::domain_to_ascii(&domain).ok()?;
    is_valid_ascii_hostname(&ascii).then_some(ascii)
}

fn is_valid_ascii_hostname(domain: &str) -> bool {
    if domain.is_empty() || domain.len() > 253 {
        return false;
    }

    domain.split('.').all(|label| {
        !label.is_empty()
            && label.len() <= 63
            && !label.starts_with('-')
            && !label.ends_with('-')
            && label
                .bytes()
                .all(|byte| byte.is_ascii_lowercase() || byte.is_ascii_digit() || byte == b'-')
    })
}
