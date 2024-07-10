var url = "https://createmod.com"
const host = window.location.host;
if (host === "127.0.0.1:8090") {
    url = "http://127.0.0.1:8090"
}
const pb = new PocketBase(url)

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

let run = function () {
    if (isAuthenticated(ignore)) {
        let loginButton = document.getElementById("login-button")
        loginButton.innerText = "Logout"
        loginButton.onclick = (ev) => {
            pb.authStore.clear();
            location.href = '/login'
        }
    }

    // Login Handler
    let loginForm = document.getElementById("login-form");
    if (loginForm != null) {
        let username = document.getElementById("username");
        let password = document.getElementById("password");
        let loginSuccess = document.getElementById("success");
        let loginError = document.getElementById("error");
        var errorDivs = [];

        loginForm.addEventListener("submit", async (e) => {
            e.preventDefault();
            var errors = [];
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
                        var div = document.createElement('div');
                        div.classList.add("invalid-feedback")
                        div.innerText = error
                        password.parentNode.insertAdjacentElement("beforeend", div)
                        errorDivs.push(div)
                    });
                })
            } else {
                errors.forEach((error) => {
                    var div = document.createElement('div');
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
                let loginError = document.getElementById("error");
                if (loginError != null) {
                    loginError.classList.remove("hidden")
                    loginError.classList.add("flex")
                }
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
                let loginError = document.getElementById("error");
                if (loginError != null) {
                    loginError.classList.remove("hidden")
                    loginError.classList.add("flex")
                }
            });
        })
    }

    // TODO check everything below this

    // Signup Handler
    let signupForm = document.getElementById("signup-form");
    if (signupForm != null) {
        let username = document.getElementById("username");
        let password = document.getElementById("password");
        let email = document.getElementById("email");
        let signupSuccess = document.getElementById("success");
        let signupError = document.getElementById("error");
        signupForm.addEventListener("submit", async (e) => {
            e.preventDefault();
            signupError.classList.remove("flex")
            signupError.classList.add("hidden")
            if (username.value === "" || password.value === "" || email.value === "") {
                signupError.classList.remove("hidden")
                signupError.classList.add("flex")
            } else {
                const data = {
                    "username": username.value,
                    "email": email.value,
                    "emailVisibility": false,
                    "password": password.value,
                    "passwordConfirm": password.value
                };

                pb.collection('users').create(data).then((record) => {
                    signupSuccess.classList.remove("hidden")
                    signupSuccess.classList.add("flex")
                    pb.collection('users').requestVerification(email.value);
                }).catch(() => {
                    signupError.classList.remove("hidden")
                    signupError.classList.add("flex")
                });
            }
        });
    }

    // Forgot Password Handler
    let forgotPasswordForm = document.getElementById("forgot-password-form");
    if (forgotPasswordForm != null) {
        let email = document.getElementById("email");
        let forgotPasswordSuccess = document.getElementById("success");
        let forgotPasswordError = document.getElementById("error");
        forgotPasswordForm.addEventListener("submit", async (e) => {
            e.preventDefault();
            forgotPasswordError.classList.remove("flex")
            forgotPasswordError.classList.add("hidden")
            if (email.value === "") {
                forgotPasswordError.classList.remove("hidden")
                forgotPasswordError.classList.add("flex")
            } else {
                pb.collection('users').requestPasswordReset(email.value).then((record) => {
                    forgotPasswordSuccess.classList.remove("hidden")
                    forgotPasswordSuccess.classList.add("flex")
                }).catch(() => {
                    forgotPasswordError.classList.remove("hidden")
                    forgotPasswordError.classList.add("flex")
                });
            }
        });
    }
}

// Run everything
run()