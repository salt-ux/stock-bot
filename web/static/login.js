const registerForm = document.getElementById("register-form");
const loginForm = document.getElementById("login-form");
const message = document.getElementById("message");

async function submitAuth(
  url,
  payload,
  pendingText,
  fallbackError,
  okClass = "ok",
  redirectTo = ""
) {
  message.className = "message";
  message.textContent = pendingText;

  try {
    const res = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });

    let data = {};
    try {
      data = await res.json();
    } catch (_error) {
      data = {};
    }
    if (!res.ok) {
      message.classList.add("error");
      message.textContent = data.message || fallbackError;
      return;
    }

    message.classList.add(okClass);
    message.textContent = data.message || "성공";

    const target = data.redirect_to || redirectTo;
    if (target) {
      window.location.assign(target);
    }
  } catch (error) {
    message.classList.add("error");
    message.textContent = "서버 통신 중 오류가 발생했습니다.";
  }
}

registerForm.addEventListener("submit", async (event) => {
  event.preventDefault();

  const formData = new FormData(registerForm);
  const id = String(formData.get("id") || "").trim();
  const password = String(formData.get("password") || "");

  await submitAuth(
    "/auth/register",
    { id, password },
    "회원가입 처리 중...",
    "회원가입에 실패했습니다."
  );
});

loginForm.addEventListener("submit", async (event) => {
  event.preventDefault();

  const formData = new FormData(loginForm);
  const id = String(formData.get("id") || "").trim();
  const password = String(formData.get("password") || "");

  await submitAuth(
    "/auth/login",
    { id, password },
    "로그인 확인 중...",
    "로그인에 실패했습니다.",
    "ok",
    "/"
  );
});
