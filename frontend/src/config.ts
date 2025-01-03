import { useMediaQuery } from "@vueuse/core";

const baseUrl: string =
  window.location.pathname.split("/").slice(0, -1).join("/") + "/";

const isXS = useMediaQuery("(max-width: 575.98px)");
const isSM = useMediaQuery("(min-width: 576px) and (max-width: 767.98px)");
const isMD = useMediaQuery("(min-width: 768px) and (max-width: 991.98px)");
const isLG = useMediaQuery("(min-width: 992px) and (max-width: 1199.98px)");
const isXL = useMediaQuery("(min-width: 1200px) and (max-width: 1399.98px)");
const isXXL = useMediaQuery("(min-width: 1400px)");

export default {
  baseUrl,
  isXS,
  isSM,
  isMD,
  isLG,
  isXL,
  isXXL,
};
