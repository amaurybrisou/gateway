import React from "react";

function Service({ name, description, image, has_access, status, is_free }) {
  const getStatusColor = () => {
    if (status === "OK") {
      return "bg-green-300";
    } else {
      return "bg-red-400";
    }
  };

  const isActive = status === "OK";

  const renderFreeBanner = () => {
    if (is_free) {
      return (
        <div className="absolute bottom-0 right-0 translate-x-8 -translate-y-6 -rotate-45">
          <div className="bg-white text-gray-800 py-1 px-8 text-sm">
            Free Service
          </div>
        </div>
      );
    }
    return null;
  };

  const displayService = () => {
    return (
      <div className="service bg-white shadow-lg rounded-lg overflow-hidden">
        <div className="relative">
          <img className="w-full h-auto object-cover" src={image} alt={name} />
          <div className="absolute top-0 left-0 p-4">
            <h2 className="text-gray-800 text-xl font-bold capitalize">{name}</h2>
          </div>
          <span
            data-tooltip-target="status-tooltip"
            className={`absolute top-5 right-4 w-5 h-5 rounded-full ${getStatusColor()}`}
          ></span>
          <div
            id="status-tooltip"
            role="tooltip"
            className="absolute z-10 invisible inline-block px-3 py-2 text-sm font-medium text-white bg-gray-900 rounded-lg shadow-sm opacity-0 tooltip dark:bg-gray-700"
          >
            {status}
            <div className="tooltip-arrow" data-popper-arrow></div>
          </div>
          {renderFreeBanner()}
        </div>
        <div className="p-4">
          <p className="text-gray-700">{description}</p>
        </div>
      </div>
    );
  };

  return (
    (isActive && <a href={`${window.location}api/${has_access ? "" : "pricing/"}${name}`}>
      {displayService()}
    </a>) || displayService()
  );
}

export default Service;
