import React, { useEffect, useState } from 'react';
import { getServices } from '../svc/svc';
import Service from './Service';

const  HomePage =  () => {
 const [services, setServices] = useState([])

 useEffect( () => {
    getServices().then(setServices).catch(console.log)
 }, [])

  return (
    <div>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="py-8">
          <h2 className="text-2xl font-bold">Services</h2>
          {/* Replace the following placeholder with your Service components */}
          <div className="mt-4 grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
            {services &&
              services.map((s) => (
                <Service
                  key={s.id}
                  name={s.name}
                  description={s.description}
                  image={s.image_url}
                  has_access={s.has_access}
                  status={s.status}
                  is_free={s.is_free}
                />
              ))}
          </div>
        </div>
      </div>
    </div>
  );
}

export default HomePage;
