import * as React from "react";
import { Action } from "../actions";
import ServiceSelectDropdown from "./ServiceSelectDropdown";
import CalculateSelector from "./CalculateSelector";

interface HomeProps {}

const Home = (props: HomeProps) => {
  return (
    <div>
      <ServiceSelectDropdown />
      <CalculateSelector />
    </div>
  );
};

export default Home;
