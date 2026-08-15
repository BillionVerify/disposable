use disposable::{count, is_domain, is_email, DisposableDomains};

#[test]
fn embedded_domains_support_normalized_exact_lookup() {
    assert!(is_domain("mailinator.com"));
    assert!(is_domain("  MAILINATOR.com  "));
    assert!(!is_domain("example.com"));
    assert!(count() >= 100_000);
}

#[test]
fn email_lookup_extracts_the_domain_and_rejects_malformed_input() {
    assert!(is_email("user@mailinator.com"));
    assert!(!is_email("alice@example.com"));
    assert!(!is_email("not-an-email"));
    assert!(!is_email("@mailinator.com"));
    assert!(!is_email("alice@"));
}

#[test]
fn custom_lists_apply_exception_exact_and_wildcard_precedence() {
    let domains = DisposableDomains::from_lists(
        "trashy.example\nalso-bad.example\n",
        "*.tempmail.example\n",
        "trashy.example\n",
    );

    assert!(!domains.is_domain("trashy.example"));
    assert!(domains.is_domain("also-bad.example"));
    assert!(domains.is_domain("tempmail.example"));
    assert!(domains.is_domain("a.b.tempmail.example"));
    assert!(!domains.is_domain("mailinator.com"));
    assert!(domains.is_email("user@a.tempmail.example"));
    assert_eq!(domains.count(), 2);
}

#[test]
fn custom_lists_normalize_idns_and_drop_invalid_hostnames() {
    let domains = DisposableDomains::from_lists(
        "Möller.example\n404: not found.example\n",
        "*.Bücher.example\n",
        "",
    );

    assert!(domains.is_domain("möller.example"));
    assert!(domains.is_domain("xn--mller-jua.example"));
    assert!(domains.is_domain("shop.bücher.example"));
    assert!(domains.is_domain("xn--bcher-kva.example"));
    assert!(!domains.is_domain("404: not found.example"));
    assert!(!domains.is_domain("bad_label.example"));
    assert_eq!(domains.count(), 1);
}
