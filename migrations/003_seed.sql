INSERT INTO entity_store(kind,id,version,payload)
VALUES
('placement','plc_home_hero',1,'{"id":"plc_home_hero","key":"home.hero","name":"首页主推荐位","capacity":5,"version":1}'::jsonb),
('audience','aud_all',1,'{"id":"aud_all","name":"全部访客","attributes":{}}'::jsonb)
ON CONFLICT(kind,id) DO NOTHING;

