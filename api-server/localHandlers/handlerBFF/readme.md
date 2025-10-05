# Backend For Frontend (BFF) Security Framework

<https://docs.duendesoftware.com/bff/>
<https://auth0.com/blog/the-backend-for-frontend-pattern-bff/>

## Go packages
<https://www.talentica.com/blogs/backend-for-frontend-bff-authentication-what-it-is-and-how-to-implement-it-in-go/>
<https://www.talentica.com/blogs/backend-for-frontend-bff-authentication-in-go-part-2/>
<https://dev.to/mehulgohil/backend-for-frontend-authentication-in-go-2e98>
<https://github.com/adityaeka26/go-bff>



 Set the session cookie to strict and http only and secure

## CSRF attack protection
Effective CSRF attack protection relies on these pillars:

Using Same-Site=strict Cookies
Requiring a specific header to be sent on every API request (IE: x-csrf=1)
having a cors policy that restricts the cookies only to a list of white-listed origins.

