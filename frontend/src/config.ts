const baseUrl: string =
  window.location.pathname.split("/").slice(0, -1).join("/") + "/";

export default {
  baseUrl,
};
