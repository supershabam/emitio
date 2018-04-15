import * as React from "react";
import { Action } from "../actions";
import ServiceSelectDropdown from "../components/ServiceSelectDropdown";

interface HomeProps {
  services: string[];
  selected: string;
  select(selected: String): Action;
}

const Home = (props: HomeProps) => {
  const { services, selected, select } = props;
  console.log("home props", props);
  return (
    <ServiceSelectDropdown
      services={services}
      selected={selected}
      select={(service: string): void => {
        select(service);
      }}
    />
  );
};

export default Home;
