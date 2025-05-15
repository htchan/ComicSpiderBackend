--
-- PostgreSQL database dump
--

-- Dumped from database version 17.5
-- Dumped by pg_dump version 17.5

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: user_websites; Type: TABLE; Schema: public; Owner: web_history
--

CREATE TABLE public.user_websites (
    website_uuid character varying(64),
    user_uuid character varying(64),
    access_time timestamp without time zone,
    group_name text
);


ALTER TABLE public.user_websites OWNER TO web_history;

--
-- Name: websites; Type: TABLE; Schema: public; Owner: web_history
--

CREATE TABLE public.websites (
    uuid character varying(64),
    url text,
    title text,
    content text,
    update_time timestamp without time zone
);


ALTER TABLE public.websites OWNER TO web_history;

--
-- Name: user_websites__user_and_uuid; Type: INDEX; Schema: public; Owner: web_history
--

CREATE UNIQUE INDEX user_websites__user_and_uuid ON public.user_websites USING btree (user_uuid, website_uuid);


--
-- Name: websites__url; Type: INDEX; Schema: public; Owner: web_history
--

CREATE UNIQUE INDEX websites__url ON public.websites USING btree (url);


--
-- Name: websites__uuid; Type: INDEX; Schema: public; Owner: web_history
--

CREATE UNIQUE INDEX websites__uuid ON public.websites USING btree (uuid);


--
-- PostgreSQL database dump complete
--

