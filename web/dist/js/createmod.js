let url = "https://createmod.com"
const host = window.location.host;
if (host === "127.0.0.1:8090") {
    url = "http://127.0.0.1:8090"
} else if (host === "0.0.0.0:8090") {
    url = "http://0.0.0.0:8090"
}
const pb = new PocketBase(url)
const acceptedCookies = getCookie("cookie_accept")

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
                _callback(false)
            }
        }).catch(() => {
            pb.authStore.clear()
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
                        // TODO a default icon could be added
                        authAvatars.item(i).remove()
                    } else {
                        authAvatars.item(i).style.backgroundImage = "url('" + pb.authStore.model.avatar + "')"
                    }
                }
                for (let i = 0; i < authUsernames.length; i++) {
                    authUsernames.item(i).innerText = pb.authStore.model.username
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

// Run everything
run()