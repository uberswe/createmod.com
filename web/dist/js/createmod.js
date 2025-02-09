let url = "https://createmod.com"
const host = window.location.host;
if (host === "127.0.0.1:8090") {
    url = "http://127.0.0.1:8090"
} else if (host === "0.0.0.0:8090") {
    url = "http://0.0.0.0:8090"
}
const pb = new PocketBase(url)
const acceptedCookies = getCookie("cookie_accept")
const dropdownSvg = '<svg  xmlns="http://www.w3.org/2000/svg"  width="24"  height="24"  viewBox="0 0 24 24"  fill="currentColor"  class="icon icon-tabler icons-tabler-filled icon-tabler-caret-down"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M18 9c.852 0 1.297 .986 .783 1.623l-.076 .084l-6 6a1 1 0 0 1 -1.32 .083l-.094 -.083l-6 -6l-.083 -.094l-.054 -.077l-.054 -.096l-.017 -.036l-.027 -.067l-.032 -.108l-.01 -.053l-.01 -.06l-.004 -.057v-.118l.005 -.058l.009 -.06l.01 -.052l.032 -.108l.027 -.067l.07 -.132l.065 -.09l.073 -.081l.094 -.083l.077 -.054l.096 -.054l.036 -.017l.067 -.027l.108 -.032l.053 -.01l.06 -.01l.057 -.004l12.059 -.002z" /></svg>'

if (acceptedCookies !== "true") {
    console.log(acceptedCookies)
    let cookiePrompt = document.getElementById("offcanvasBottom")
    cookiePrompt.classList.remove("hide")
    cookiePrompt.classList.add("show")
}

let isAuthenticated = function (_callback) {
    if (pb.authStore.isValid) {
        authRefresh().then(() => {
            if (pb.authStore.isValid) {
                _callback(true)
            } else {
                pb.authStore.clear()
                // clear cookie
                deleteCookie("create-mod-auth")
                _callback(false)
            }
        }).catch(() => {
            pb.authStore.clear()
            deleteCookie("create-mod-auth")
            _callback(false)
        })
        return true
    }
    return false
}

let authRefresh = async function () {
    return pb.collection('users').authRefresh();
}

function ignore(loggedIn) {
    // do nothing
}

function acceptCookies() {
    setCookie("cookie_accept", true, 365)
}

function setCookie(c_name, value, exdays) {
    var exdate = new Date();
    exdate.setDate(exdate.getDate() + exdays);
    var c_value = escape(value) + ((exdays == null) ? "" : "; expires=" + exdate.toUTCString());
    document.cookie = c_name + "=" + c_value;
}

function getCookie(c_name) {
    var i, x, y, ARRcookies = document.cookie.split(";");
    for (i = 0; i < ARRcookies.length; i++) {
        x = ARRcookies[i].substr(0, ARRcookies[i].indexOf("="));
        y = ARRcookies[i].substr(ARRcookies[i].indexOf("=") + 1);
        x = x.replace(/^\s+|\s+$/g, "");
        if (x == c_name) {
            return unescape(y);
        }
    }
}

function deleteCookie( name, path, domain ) {
    if( getCookie( name ) ) {
        document.cookie = name + "=" +
            ((path) ? ";path="+path:"")+
            ((domain)?";domain="+domain:"") +
            ";expires=Thu, 01 Jan 1970 00:00:01 GMT";
    }
}

let run = function () {
    if (isAuthenticated(ignore)) {
        let loginButton = document.getElementById("login-button")
        if (loginButton != null) {
            loginButton.innerText = "Logout"
            loginButton.onclick = (ev) => {
                pb.authStore.clear();
                location.href = '/login'
            }
        }
    }

    // Login Handler
    let loginForm = document.getElementById("login-form");
    if (loginForm != null) {
        let username = document.getElementById("username");
        let password = document.getElementById("password");
        let errorDivs = [];

        loginForm.addEventListener("submit", async (e) => {
            e.preventDefault();
            let errors = [];
            errorDivs.forEach((ed) => {
                ed.remove()
            })
            errorDivs = []
            username.classList.remove("is-invalid")
            password.classList.remove("is-invalid")
            if (username.value === "") {
                username.classList.add("is-invalid")
                errors.push("Invalid email")
            }
            if (password.value === "") {
                password.classList.add("is-invalid")
                errors.push("Invalid password")
            }
            if (errors.length === 0) {
                pb.collection('users').authWithPassword(
                    username.value,
                    password.value,
                ).then((authData) => {
                    location.href = '/'
                }).catch((e) => {
                    password.classList.add("is-invalid")
                    errors.push("Invalid password or the account does not exist.")
                }).finally(() => {
                    errors.forEach((error) => {
                        let div = document.createElement('div');
                        div.classList.add("invalid-feedback")
                        div.innerText = error
                        password.parentNode.insertAdjacentElement("beforeend", div)
                        errorDivs.push(div)
                    });
                })
            } else {
                errors.forEach((error) => {
                    let div = document.createElement('div');
                    div.classList.add("invalid-feedback")
                    div.innerText = error
                    password.parentNode.insertAdjacentElement("beforeend", div)
                    errorDivs.push(div)
                });
            }
        });
    }

    // Discord login
    let discordLogin = document.getElementById("discord-login");
    if (discordLogin != null) {
        discordLogin.addEventListener("click", async (e) => {
            e.preventDefault();
            await pb.collection('users').authWithOAuth2({provider: 'discord'}).then(() => {
                location.href = '/'
            }).catch(() => {
                // Throw some error
            });
        })
    }

    // Github login
    let githubLogin = document.getElementById("github-login");
    if (githubLogin != null) {
        githubLogin.addEventListener("click", async (e) => {
            e.preventDefault();
            await pb.collection('users').authWithOAuth2({provider: 'github'}).then(() => {
                location.href = '/'
            }).catch(() => {
                // Throw some error
            });
        })
    }

    // Signup Handler
    let signupForm = document.getElementById("signup-form");
    if (signupForm != null) {
        let username = document.getElementById("username");
        let password = document.getElementById("password");
        let email = document.getElementById("email");
        let terms = document.getElementById("terms");
        let errorDivs = [];
        signupForm.addEventListener("submit", async (e) => {
                e.preventDefault();
                let errors = [];
                errorDivs.forEach((ed) => {
                    ed.remove()
                })
                errorDivs = []
                username.classList.remove("is-invalid")
                password.classList.remove("is-invalid")
                email.classList.remove("is-invalid")
                terms.classList.remove("is-invalid")
                if (username.value === "") {
                    username.classList.add("is-invalid")
                    errors.push("Invalid username")
                }
                if (password.value === "") {
                    password.classList.add("is-invalid")
                    errors.push("Invalid password")

                }
                if (email.value === "") {
                    email.classList.add("is-invalid")
                    errors.push("Invalid email")
                }
                if (!terms.checked) {
                    terms.classList.add("is-invalid")
                    errors.push("You must agree to the Terms Of Service")
                }
                if (errors.length === 0) {
                    const data = {
                        "username": username.value,
                        "email": email.value,
                        "emailVisibility": false,
                        "password": password.value,
                        "passwordConfirm": password.value,
                        "terms": terms.checked
                    };

                    pb.collection('users').create(data).then((record) => {
                        pb.collection('users').requestVerification(email.value);
                        let successModal = new bootstrap.Modal(document.getElementById('modal-success'), {});
                        successModal.show();
                    }).catch((e) => {
                        for (const [key, value] of Object.entries(e.data.data)) {
                            let div = document.createElement('div');
                            div.classList.add("invalid-feedback")
                            div.innerText = value.message
                            let element = document.getElementById(key);
                            element.classList.add("is-invalid")
                            element.parentNode.insertAdjacentElement("beforeend", div)
                            errorDivs.push(div)
                        }
                    });
                } else {
                    errors.forEach((error) => {
                        let div = document.createElement('div');
                        div.classList.add("invalid-feedback")
                        div.innerText = error
                        password.parentNode.insertAdjacentElement("beforeend", div)
                        errorDivs.push(div)
                    });
                }
            }
        )
        ;
    }

// Forgot Password Handler
    let forgotPasswordForm = document.getElementById("forgot-password-form");
    if (forgotPasswordForm != null) {
        let div = document.createElement('div');
        let email = document.getElementById("email");
        forgotPasswordForm.addEventListener("submit", async (e) => {
            e.preventDefault();
            div.remove()
            if (email.value === "") {
                email.classList.add("is-invalid")
                div = document.createElement('div');
                div.classList.add("invalid-feedback")
                div.innerText = "Invalid email"
                email.parentNode.insertAdjacentElement("beforeend", div)
            } else {
                pb.collection('users').requestPasswordReset(email.value).then((record) => {
                    let successModal = new bootstrap.Modal(document.getElementById('modal-success'), {});
                    successModal.show();
                }).catch((e) => {
                    email.classList.add("is-invalid")
                    div = document.createElement('div');
                    div.classList.add("invalid-feedback")
                    div.innerText = e.data.data.email.message
                    email.parentNode.insertAdjacentElement("beforeend", div)
                });
            }
        });
    }

    let logoutButtons = document.getElementsByClassName("logout-button");
    if (logoutButtons != null) {

        for (let i = 0; i < logoutButtons.length; i++) {
            logoutButtons.item(i).addEventListener("click", async (e) => {
                pb.authStore.clear();
                deleteCookie("create-mod-auth")
                location.href = '/'
            })
        }
    }

    let authDropdowns = document.getElementsByClassName("auth-dropdown")
    function renderDropdown(isLoggedIn) {
        if (isLoggedIn) {
            let authAvatars = document.getElementsByClassName("auth-avatar")
            let authUsernames = document.getElementsByClassName("auth-username")
            if (authAvatars != null && authUsernames != null) {

                for (let i = 0; i < authAvatars.length; i++) {
                    if (pb.authStore.model.avatar === "") {
                        authAvatars.item(i).remove()
                    } else {
                        authAvatars.item(i).style.backgroundImage = "url('" + pb.authStore.model.avatar + "')"
                    }
                }
                console.log(pb.authStore.model)
                for (let i = 0; i < authUsernames.length; i++) {
                    authUsernames.item(i).innerHTML = pb.authStore.model.username + " " + dropdownSvg
                }
            }
        } else {
            for (let i = 0; i < authDropdowns.length; i++) {
                authDropdowns.item(i).innerHTML = "<a href=\"/login\" >Login</a>"
            }
        }
    }

    if (authDropdowns != null && authDropdowns.length !== 0) {
        renderDropdown(isAuthenticated(renderDropdown))
    }

    let searchForm = document.getElementById("search-form")
    let searchField = document.getElementById("search-field")
    if (searchForm != null && searchField != null) {
        searchForm.addEventListener("submit", async (e) => {
            e.preventDefault();
            let slug = slugify(searchField.value)
            location.href = "/search/" + slug
        })
    }

    function slugify(str) {
        str = str.replace(/^\s+|\s+$/g, ''); // trim leading/trailing white space
        str = str.toLowerCase(); // convert string to lowercase
        str = str.replace(/[^a-z0-9 -]/g, '') // remove any non-alphanumeric characters
            .replace(/\s+/g, '-') // replace spaces with hyphens
            .replace(/-+/g, '-'); // remove consecutive hyphens
        return str;
    }
}

document.addEventListener("DOMContentLoaded", function() {
// Run everything
    run()
})