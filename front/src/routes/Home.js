
import { useEffect, useState } from 'react';
import Service from '../components/service';
import { ACCESS_TOKEN_KEY, API_URL } from '../constants';

const getServices = async () => {
  var headers;
  const token = sessionStorage.getItem(ACCESS_TOKEN_KEY);
  if (token) {
    headers = {
      Authorization: "Bearer " + token,
    }
  }
  const response = await fetch(API_URL + '/services', {    headers  });

  if (response.ok) {
    const data = await response.json();
    return data;
  }

  return [];
};

export default function Home() {
 const [services, setServices] =  useState([])

 useEffect(()=>{
  getServices().then(setServices)
 }, [])

  return (
    <div>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="py-8">
          <h2 className="text-2xl font-bold">Services</h2>
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

