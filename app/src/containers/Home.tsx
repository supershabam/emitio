import * as React from "react";
import { Action } from "../actions";
import ServiceSelectDropdown from "./ServiceSelectDropdown";

interface HomeProps {}

const Home = (props: HomeProps) => {
  return <ServiceSelectDropdown />;
};

export default Home;
