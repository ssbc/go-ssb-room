+++
{
    "notice": "NoticePrivacyPolicy",
    "title": "Privacy Policy",
    "language": "en-GB"
}
+++

## We are not interested in user data

We are not interested in tracking user data or metadata for profit or for building personally identifiable "profiles".

This room server, by design, does not replicate any SSB feeds whatsoever. If we do not host user content, we cannot track that kind of data at all.

We also do not persistently store IP addresses of members or visitors.

## Exceptions

There is only minimal amount of information that this server knows of its members and visitors, which are:

- Each member's SSB ID is stored in a database. This is necessary to grant access to members only
- Each member's "alias" URL is stored in a database. This is necessary to make aliases work, but not all members need to have aliases

## External service to check for leaked passwords

We use the external service of [haveibeenpwned.com (HIBP)](https://haveibeenpwned.com/Passwords) to check if a member's login password is contained in a known data leak, making them susceptible to a [credential stuffing](https://en.wikipedia.org/wiki/Credential_stuffing) attack. Since we only send a subset of the hashed password to HIBP, the actual password is _not_ sent to HIBP, nor is any other member information. The technique is explained in more detail in [this blog article](https://www.troyhunt.com/ive-just-launched-pwned-passwords-version-2/#cloudflareprivacyandkanonymity).
We list this here in the interests of transparency, since an error message indicating the use of the HIBP service will be displayed if there is an attempt to use a leaked password. The HIBP service is not used for any member who solely uses _Sign-In with SSB_ and not the password-based login.
