* Notes



```go
            document.getElementById("fetchDataBtn").addEventListener("click", () => {
                const userData = fetchUserData();
                userData.then(data => {
                    const user = JSON.parse(data);
                    document.getElementById("output").innerHTML = `
                        <p>ID: ${user.id}</p>
                        <p>Name: ${user.name}</p>
                        <p>Username: ${user.username}</p>
                        <p>Email: ${user.email}</p>
                    `;
                });
            });

        });
```